package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"nun/internal/nixexpr"
)

type App struct {
	Root string
}

type PackageEntry struct {
	Set  string
	Name string
}

type PendingWrite struct {
	Path         string
	RelativePath string
	Content      string
}

type NewHostDefaults struct {
	DefaultName     string
	DefaultUser     string
	DefaultSystem   string
	ExistingHosts   []string
	PackageSetNames []string
}

type NewHostRequest struct {
	Name        string
	User        string
	System      string
	PackageSets []string
}

type HostPlan struct {
	Name        string
	User        string
	System      string
	PackageSets []string
	Writes      []PendingWrite
}

func NewApp() App {
	return App{}
}

func (a App) repoRoot() (string, error) {
	if a.Root != "" {
		return a.Root, nil
	}
	if root := os.Getenv("NUN_CONFIG_ROOT"); root != "" {
		return root, nil
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, "nix-config")
		if _, err := os.Stat(filepath.Join(candidate, "flake.nix")); err == nil {
			return candidate, nil
		}
	}
	return os.Getwd()
}

func (a App) ReadPackageSets() ([]PackageEntry, error) {
	root, err := a.repoRoot()
	if err != nil {
		return nil, err
	}
	files, err := nixexpr.ReadPackageSetFiles(filepath.Join(root, "package-sets"))
	if err != nil {
		return nil, err
	}
	var entries []PackageEntry
	for _, file := range files {
		for _, name := range file.Packages {
			entries = append(entries, PackageEntry{Set: file.Name, Name: name})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Set == entries[j].Set {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Set < entries[j].Set
	})
	return entries, nil
}

func (a App) NewHostDefaults() (NewHostDefaults, error) {
	root, err := a.repoRoot()
	if err != nil {
		return NewHostDefaults{}, err
	}
	hosts, err := readDirNames(filepath.Join(root, "hosts"))
	if err != nil {
		return NewHostDefaults{}, err
	}
	sets, err := nixexpr.ReadPackageSetNames(filepath.Join(root, "package-sets"))
	if err != nil {
		return NewHostDefaults{}, err
	}
	name, _ := os.Hostname()
	system := "x86_64-linux"
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		system = "aarch64-darwin"
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}
	return NewHostDefaults{
		DefaultName:     name,
		DefaultUser:     user,
		DefaultSystem:   system,
		ExistingHosts:   hosts,
		PackageSetNames: sets,
	}, nil
}

func (a App) HostNames() ([]string, error) {
	root, err := a.repoRoot()
	if err != nil {
		return nil, err
	}
	return readDirNames(filepath.Join(root, "hosts"))
}

func (a App) PlanNewHost(req NewHostRequest) (HostPlan, error) {
	root, err := a.repoRoot()
	if err != nil {
		return HostPlan{}, err
	}
	if err := validateHostRequest(req, root); err != nil {
		return HostPlan{}, err
	}
	hostDir := filepath.Join(root, "hosts", req.Name)
	writes := []PendingWrite{{
		Path:         filepath.Join(hostDir, "default.nix"),
		RelativePath: filepath.Join("hosts", req.Name, "default.nix"),
		Content: RenderHostDefault(HostDefaultInput{
			Name:             req.Name,
			User:             req.User,
			System:           req.System,
			ConfigRoot:       filepath.Join("/Users", req.User, "nix-config"),
			HasPackageModule: len(req.PackageSets) > 0,
		}),
	}}
	if len(req.PackageSets) > 0 {
		writes = append(writes, PendingWrite{
			Path:         filepath.Join(hostDir, "packages.nix"),
			RelativePath: filepath.Join("hosts", req.Name, "packages.nix"),
			Content:      RenderHostPackages(req.User, req.PackageSets),
		})
	}
	return HostPlan{
		Name:        req.Name,
		User:        req.User,
		System:      req.System,
		PackageSets: append([]string(nil), req.PackageSets...),
		Writes:      writes,
	}, nil
}

func (a App) ApplyWrites(writes []PendingWrite, out io.Writer) error {
	for _, write := range writes {
		if err := os.MkdirAll(filepath.Dir(write.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(write.Path, []byte(write.Content), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "created %s\n", write.RelativePath)
	}
	return nil
}

func (a App) PrintCurrentHost(out io.Writer) error {
	root, err := a.repoRoot()
	if err != nil {
		return err
	}
	current, _ := os.Hostname()
	hosts, err := readDirNames(filepath.Join(root, "hosts"))
	if err != nil {
		return err
	}
	marker := " (not configured)"
	for _, host := range hosts {
		if host == current {
			marker = ""
			break
		}
	}
	fmt.Fprintf(out, "%s%s\n", current, marker)
	return nil
}

func PrintHostPlan(out io.Writer, plan HostPlan) {
	fmt.Fprintln(out, "\nPlan:")
	fmt.Fprintf(out, "  Create host: %s\n", plan.Name)
	fmt.Fprintf(out, "  Primary user: %s\n", plan.User)
	fmt.Fprintf(out, "  System: %s\n", plan.System)
	sets := "none"
	if len(plan.PackageSets) > 0 {
		sets = strings.Join(plan.PackageSets, ", ")
	}
	fmt.Fprintf(out, "  Package sets: %s\n", sets)
	fmt.Fprintln(out, "  Files to add:")
	for _, write := range plan.Writes {
		fmt.Fprintf(out, "    %s\n", write.RelativePath)
	}
	fmt.Fprintln(out)
}

func (a App) Rebuild(args []string) error {
	var host string
	remote := false
	var forwarded []string
	for _, arg := range args {
		if arg == "--remote" {
			remote = true
		} else if host == "" && !strings.HasPrefix(arg, "-") {
			host = arg
		} else {
			forwarded = append(forwarded, arg)
		}
	}

	localHost, _ := os.Hostname()
	if host == "" {
		host = localHost
	}
	if remote && host == "" {
		return fmt.Errorf("hostname not specified for remote build")
	}
	if !remote && host != localHost {
		fmt.Fprintf(os.Stderr, "warn: building local configuration for hostname %q, but local hostname is %q\n", host, localHost)
	}
	if remote {
		return a.rebuildRemote(host, forwarded)
	}

	root, err := a.repoRoot()
	if err != nil {
		return err
	}
	separator := indexOf(forwarded, "--")
	nhFlags := forwarded
	var nixFlags []string
	if separator >= 0 {
		nhFlags = forwarded[:separator]
		nixFlags = forwarded[separator+1:]
	}
	osArgs := []string{"os", "switch", "."}
	env := os.Environ()
	if runtime.GOOS == "darwin" {
		osArgs = []string{"darwin", "switch", "."}
	} else {
		env = append(env, "NH_BYPASS_ROOT_CHECK=true")
	}
	cmd := append([]string{"nh"}, osArgs...)
	cmd = append(cmd, "--hostname", host)
	cmd = append(cmd, nhFlags...)
	cmd = append(cmd, "--", "--accept-flake-config", "--extra-experimental-features", "pipe-operators")
	cmd = append(cmd, nixFlags...)
	return run(root, env, cmd...)
}

func (a App) rebuildRemote(host string, forwarded []string) error {
	root, err := a.repoRoot()
	if err != nil {
		return err
	}
	if err := run("", nil, "ssh", "-tt", "root@"+host, "rm --recursive --force ncc"); err != nil {
		return err
	}
	files, err := exec.Command("git", "-C", root, "ls-files").Output()
	if err != nil {
		return err
	}
	cmd := exec.Command("rsync", "--archive", "--compress", "--delete", "--recursive", "--force", "--delete-excluded", "--delete-missing-args", "--human-readable", "--delay-updates", "--files-from", "-", root, "root@"+host+":ncc")
	cmd.Stdin = bytes.NewReader(files)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	remoteCmd := "cd ncc && nun rebuild " + shellJoin(append([]string{host}, forwarded...))
	return run("", nil, "ssh", "-tt", "root@"+host, remoteCmd)
}

func validateHostRequest(req NewHostRequest, root string) error {
	if !nixexpr.ValidIdentifier(req.Name) {
		return fmt.Errorf("host name may only contain letters, numbers, '_' and '-'")
	}
	if req.User == "" || req.System == "" {
		return fmt.Errorf("user and system cannot be empty")
	}
	if _, err := os.Stat(filepath.Join(root, "hosts", req.Name)); err == nil {
		return fmt.Errorf("host %q already exists", req.Name)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func readDirNames(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func run(cwd string, env []string, command ...string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = cwd
	if env != nil {
		cmd.Env = env
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", command[0], err)
	}
	return nil
}

func indexOf(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("_./:=+-", r))
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
