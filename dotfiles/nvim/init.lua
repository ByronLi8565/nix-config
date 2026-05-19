vim.g.mapleader = " "
vim.g.maplocalleader = " "

vim.opt.clipboard = "unnamedplus"
vim.loader.enable(true)
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.ignorecase = true
vim.opt.smartcase = true
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4
vim.opt.expandtab = true
vim.opt.termguicolors = true
vim.opt.undofile = true
vim.opt.undodir = vim.fn.stdpath("data") .. "/undo"
vim.fn.mkdir(vim.fn.stdpath("data") .. "/undo", "p")
vim.opt.scrolloff = 8
vim.opt.signcolumn = "no"
vim.opt.swapfile = false
vim.opt.autoread = true

vim.api.nvim_create_autocmd({ "FocusGained", "BufEnter" }, {
	pattern = "*",
	command = "checktime",
})

local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
	vim.fn.system({
		"git",
		"clone",
		"--filter=blob:none",
		"https://github.com/folke/lazy.nvim.git",
		"--branch=stable",
		lazypath,
	})
end
vim.opt.rtp:prepend(lazypath)

require("lazy").setup({
	{
		"folke/tokyonight.nvim",
		priority = 1000,
		config = function()
			require("tokyonight").setup({
				style = "night",
			})
			vim.cmd.colorscheme("gruvbox")
		end,
	},
	{ "ellisonleao/gruvbox.nvim", priority = 1000, config = true, opts = ... },

	{
		"nvim-treesitter/nvim-treesitter",
		build = ":TSUpdate",
		main = "nvim-treesitter.configs",
		opts = {
			ensure_installed = { "lua", "vim", "vimdoc" },
			auto_install = true,
			highlight = { enable = true },
			indent = { enable = true },
		},
	},
	{ "tpope/vim-sleuth" },
	{
		"neovim/nvim-lspconfig",
		config = function()
			vim.diagnostic.config({
				float = {
					border = "rounded",
					source = "always",
				},
			})

			vim.o.updatetime = 1000
			vim.api.nvim_create_autocmd("CursorHold", {
				callback = function()
					vim.diagnostic.open_float(nil, { focus = false })
				end,
			})
			local servers = {
				lua_ls = {
					settings = {
						Lua = {
							diagnostics = {
								globals = { "vim" },
							},
						},
					},
				},
				nil_ls = {},
			}

			for server, config in pairs(servers) do
				vim.lsp.enable(server)
				if config then
					vim.lsp.config(server, config)
				end
			end
		end,
	},
	{
		"stevearc/conform.nvim",
		config = function()
			require("conform").setup({
				formatters_by_ft = {
					lua = { "stylua" },
					nix = { "alejandra" },
				},
				format_on_save = {
					timeout_ms = 500,
					lsp_format = "fallback",
				},
			})
		end,
	},
	{
		"lervag/vimtex",
		lazy = false,
		init = function()
			vim.g.vimtex_view_method = "skim"
		end,
	},
	{
		url = "git@codeberg.org:andyg/leap.nvim.git",
		config = function()
			vim.keymap.set({ "n", "x", "o" }, "s", "<Plug>(leap-anywhere)")
			require("leap").opts.equivalence_classes = { " \t\r\n", "([{", ")]}", "'\"`" }
			require("leap").opts.preview_filter = function(ch0, ch1, ch2)
				return not (ch1:match("%s") or ch0:match("%a") and ch1:match("%a") and ch2:match("%a"))
			end
			require("leap.user").set_repeat_keys("<enter>", "<backspace>")
		end,
	},
	{
		"folke/noice.nvim",
		event = "VeryLazy",
		dependencies = {
			"MunifTanjim/nui.nvim",
			"rcarriga/nvim-notify",
		},
		config = function()
			require("noice").setup({
				lsp = {
					override = {
						["vim.lsp.util.convert_input_to_markdown_lines"] = true,
						["vim.lsp.util.stylize_markdown"] = true,
					},
				},
				messages = {
					enabled = true, -- enables the Noice messages UI
					view = "mini", -- default view for messages
					view_error = "notify", -- view for errors
					view_warn = "notify", -- view for warnings
					view_history = "messages", -- view for :messages
					view_search = "virtualtext", -- view for search count messages. Set to `false` to disable
				},
				presets = {
					command_palette = true, -- position the cmdline and popupmenu together
					long_message_to_split = true, -- long messages will be sent to a split
					inc_rename = false, -- enables an input dialog for inc-rename.nvim
					lsp_doc_border = false, -- add a border to hover docs and signature help
				},
			})
		end,
	},
})
