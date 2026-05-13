#!/usr/bin/env bun

import { readdir } from "node:fs/promises";
import { basename, extname, join } from "node:path";
import process from "node:process";
import {
  createCliRenderer,
  SelectRenderable,
  TextRenderable,
  type SelectOption,
} from "@opentui/core";

type PackageEntry = {
  set: string;
  name: string;
};

const help = `nun config helper

Usage:
  nun rebuild [host] [--remote] [nh flags...] [-- nix flags...]
  nun packages

Commands:
  rebuild   Rebuild this nix-darwin/NixOS config with nh
  packages  Browse package-sets/*.nix in an OpenTUI terminal view
`;

async function main() {
  const [command, ...args] = Bun.argv.slice(2);

  switch (command) {
    case "rebuild":
      await rebuild(args);
      break;
    case "packages":
      await packages();
      break;
    case undefined:
    case "-h":
    case "--help":
      process.stdout.write(help);
      break;
    default:
      throw new Error(`unknown command: ${command}`);
  }
}

async function rebuild(args: string[]) {
  let host: string | undefined;
  let remote = false;
  const forwarded: string[] = [];

  for (const arg of args) {
    if (arg === "--remote") {
      remote = true;
    } else if (!host && !arg.startsWith("-")) {
      host = arg;
    } else {
      forwarded.push(arg);
    }
  }

  const localHost = (await $text(["hostname"])).trim();
  const targetHost = host ?? localHost;

  if (remote && !host) {
    throw new Error("hostname not specified for remote build");
  }

  if (!remote && targetHost !== localHost) {
    process.stderr.write(
      `warn: building local configuration for hostname '${targetHost}', but local hostname is '${localHost}'\n`,
    );
  }

  if (remote) {
    await rebuildRemote(targetHost, forwarded);
    return;
  }

  const root = await repoRoot();
  const separator = forwarded.indexOf("--");
  const nhFlags = separator === -1 ? forwarded : forwarded.slice(0, separator);
  const nixFlags = separator === -1 ? [] : forwarded.slice(separator + 1);
  const osArgs =
    process.platform === "darwin"
      ? ["darwin", "switch", "."]
      : ["os", "switch", "."];

  const env =
    process.platform === "darwin"
      ? process.env
      : { ...process.env, NH_BYPASS_ROOT_CHECK: "true" };

  await run(
    [
      "nh",
      ...osArgs,
      "--hostname",
      targetHost,
      ...nhFlags,
      "--",
      "--accept-flake-config",
      "--extra-experimental-features",
      "pipe-operators",
      ...nixFlags,
    ],
    { env, cwd: root },
  );
}

async function rebuildRemote(host: string, forwarded: string[]) {
  const root = await repoRoot();
  await run(["ssh", "-tt", `root@${host}`, "rm --recursive --force ncc"]);

  const files = await $text(["git", "ls-files"], root);
  const sync = Bun.spawn([
    "rsync",
    "--archive",
    "--compress",
    "--delete",
    "--recursive",
    "--force",
    "--delete-excluded",
    "--delete-missing-args",
    "--human-readable",
    "--delay-updates",
    "--files-from",
    "-",
    root,
    `root@${host}:ncc`,
  ], {
    stdin: "pipe",
    stdout: "inherit",
    stderr: "inherit",
  });

  sync.stdin.write(files);
  sync.stdin.end();
  const syncExit = await sync.exited;
  if (syncExit !== 0) {
    throw new Error(`rsync exited with ${syncExit}`);
  }

  await run([
    "ssh",
    "-tt",
    `root@${host}`,
    `cd ncc && bun install --cwd bin/nun --frozen-lockfile && bun run --cwd bin/nun src/main.ts rebuild ${shellJoin([host, ...forwarded])}`,
  ]);
}

async function packages() {
  const entries = await readPackageSets(await repoRoot());
  if (entries.length === 0) {
    throw new Error("no packages found in package-sets/*.nix");
  }

  await runPackageTui(entries);
}

async function repoRoot() {
  if (process.env.NUN_CONFIG_ROOT) {
    return process.env.NUN_CONFIG_ROOT;
  }

  const proc = Bun.spawn(["git", "rev-parse", "--show-toplevel"], {
    stdout: "pipe",
    stderr: "ignore",
  });
  const output = await new Response(proc.stdout).text();
  if ((await proc.exited) === 0) {
    return output.trim();
  }
  return process.cwd();
}

async function readPackageSets(root: string): Promise<PackageEntry[]> {
  const packageSets = join(root, "package-sets");
  const files = (await readdir(packageSets))
    .filter((file) => extname(file) === ".nix")
    .sort();
  const entries: PackageEntry[] = [];

  for (const file of files) {
    const set = basename(file, ".nix");
    const source = await Bun.file(join(packageSets, file)).text();
    for (const name of parseNixPackageList(source)) {
      entries.push({ set, name });
    }
  }

  return entries.sort((left, right) =>
    left.set.localeCompare(right.set) || left.name.localeCompare(right.name),
  );
}

function parseNixPackageList(source: string) {
  const packages: string[] = [];
  let inList = false;

  for (const rawLine of source.split("\n")) {
    const line = rawLine.split("#", 1)[0].trim();
    if (!line) continue;
    if (line.includes("[")) {
      inList = true;
      continue;
    }
    if (line.includes("]")) break;
    if (!inList) continue;

    const packageName = line.replace(/,$/, "").trim();
    if (packageName) packages.push(packageName);
  }

  return packages;
}

async function runPackageTui(packages: PackageEntry[]) {
  const renderer = await createCliRenderer({
    clearOnShutdown: true,
    consoleMode: "disabled",
    exitOnCtrlC: false,
  });
  let filter = "";
  let selectedValue = "";

  const draw = () => {
    const visible = packages.filter((pkg) => {
      const needle = filter.toLowerCase();
      return (
        needle === "" ||
        pkg.name.toLowerCase().includes(needle) ||
        pkg.set.toLowerCase().includes(needle)
      );
    });
    const options = visible.map((pkg): SelectOption => ({
      name: pkg.name,
      description: pkg.set,
      value: `${pkg.set}:${pkg.name}`,
    }));
    const nextIndex = Math.max(
      0,
      options.findIndex((option) => option.value === selectedValue),
    );

    title.content = "nun packages";
    summary.content = `filter: ${filter}\n${visible.length} visible / ${packages.length} total packages\nj/k move  type filter  backspace delete  q quit`;
    list.options = options.length === 0
      ? [{ name: "No matching packages", description: "", value: "" }]
      : options;
    list.selectedIndex = Math.min(nextIndex, list.options.length - 1);
    selectedValue = list.getSelectedOption()?.value ?? "";
    renderer.requestRender();
  };

  const title = new TextRenderable(renderer, {
    id: "title",
    content: "nun packages",
    position: "absolute",
    top: 0,
    left: 0,
    width: renderer.terminalWidth,
    height: 1,
  });
  const summary = new TextRenderable(renderer, {
    id: "summary",
    content: "",
    position: "absolute",
    top: 2,
    left: 0,
    width: renderer.terminalWidth,
    height: 3,
  });
  const list = new SelectRenderable(renderer, {
    id: "packages",
    position: "absolute",
    top: 6,
    left: 0,
    width: renderer.terminalWidth,
    height: Math.max(1, renderer.terminalHeight - 7),
    options: [],
    showScrollIndicator: true,
    showDescription: true,
    wrapSelection: false,
    keyAliasMap: {
      j: "down",
      k: "up",
    },
  });

  renderer.root.add(title);
  renderer.root.add(summary);
  renderer.root.add(list);
  renderer.focusRenderable(list);
  renderer.start();
  draw();

  await new Promise<void>((resolve) => {
    renderer.keyInput.on("keypress", (key) => {
      if (key.ctrl && key.name === "c") {
        resolve();
        return;
      }
      if (key.name === "escape" || key.sequence === "q") {
        resolve();
        return;
      }
      if (key.name === "backspace" || key.sequence === "\u007f") {
        filter = filter.slice(0, -1);
        selectedValue = "";
        draw();
        return;
      }
      if (key.sequence.length === 1 && /^[ -~]$/.test(key.sequence)) {
        filter += key.sequence;
        selectedValue = "";
        draw();
        return;
      }

      selectedValue = list.getSelectedOption()?.value ?? "";
    });
  });

  renderer.destroy();
}

async function run(
  command: string[],
  options: { env?: NodeJS.ProcessEnv; cwd?: string } = {},
) {
  const proc = Bun.spawn(command, {
    stdin: "inherit",
    stdout: "inherit",
    stderr: "inherit",
    env: options.env ?? process.env,
    cwd: options.cwd,
  });
  const exitCode = await proc.exited;
  if (exitCode !== 0) {
    throw new Error(`${command[0]} exited with ${exitCode}`);
  }
}

async function $text(command: string[], cwd?: string) {
  const proc = Bun.spawn(command, {
    stdout: "pipe",
    stderr: "inherit",
    cwd,
  });
  const output = await new Response(proc.stdout).text();
  const exitCode = await proc.exited;
  if (exitCode !== 0) {
    throw new Error(`${command[0]} exited with ${exitCode}`);
  }
  return output;
}

function shellJoin(args: string[]) {
  return args.map(shellQuote).join(" ");
}

function shellQuote(value: string) {
  if (/^[A-Za-z0-9_./:=+-]+$/.test(value)) return value;
  return `'${value.replaceAll("'", "'\\''")}'`;
}

main().catch((error) => {
  process.stderr.write(`error: ${error.message}\n`);
  process.exit(1);
});
