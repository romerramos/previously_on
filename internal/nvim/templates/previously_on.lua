local M = {}

local executable = {{EXECUTABLE}}
local summary_buf = nil
local cached_summary = nil
local augroup = vim.api.nvim_create_augroup("PreviouslyOn", { clear = true })

local function set_lines(buf, lines)
  vim.bo[buf].modifiable = true
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
  vim.bo[buf].modifiable = false
end

local function ensure_buffer()
  if summary_buf and vim.api.nvim_buf_is_valid(summary_buf) then
    return summary_buf
  end

  summary_buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_name(summary_buf, "previously-on://summary.md")
  vim.bo[summary_buf].buftype = "nofile"
  vim.bo[summary_buf].bufhidden = "hide"
  vim.bo[summary_buf].swapfile = false
  vim.bo[summary_buf].filetype = "markdown"
  vim.bo[summary_buf].modifiable = false
  return summary_buf
end

local function show_buffer(buf)
  for _, win in ipairs(vim.api.nvim_list_wins()) do
    if vim.api.nvim_win_get_buf(win) == buf then
      vim.api.nvim_set_current_win(win)
      vim.wo[win].wrap = true
      vim.wo[win].linebreak = true
      vim.wo[win].breakindent = true
      return
    end
  end
  vim.cmd("botright split")
  vim.api.nvim_win_set_height(0, math.min(18, math.max(8, math.floor(vim.o.lines * 0.35))))
  vim.api.nvim_set_current_buf(buf)
  vim.wo.wrap = true
  vim.wo.linebreak = true
  vim.wo.breakindent = true
end

local function git_root(cwd)
  local result = vim.system({ "git", "rev-parse", "--show-toplevel" }, { cwd = cwd, text = true }):wait()
  if result.code ~= 0 then
    return nil
  end
  return vim.trim(result.stdout or "")
end

function M.open()
  local buf = ensure_buffer()
  show_buffer(buf)
  if cached_summary then
    set_lines(buf, cached_summary)
  else
    set_lines(buf, {
      "# Previously on...",
      "",
      "Loading repository summary...",
    })
  end

  local root = git_root(vim.fn.getcwd())
  if not root or root == "" then
    set_lines(buf, {
      "# Previously on...",
      "",
      "No Git repository was found for the current working directory.",
    })
    return
  end

  vim.system({ executable, "summary", "--raw", "--no-mark-seen", "--simple" }, { cwd = root, text = true }, function(result)
    vim.schedule(function()
      if not vim.api.nvim_buf_is_valid(buf) then
        return
      end
      if result.code ~= 0 then
        local stderr = vim.trim(result.stderr or "")
        if stderr == "" then
          stderr = "previously-on summary failed."
        end
        set_lines(buf, {
          "# Previously on...",
          "",
          "Failed to generate the repository summary.",
          "",
          "~~~text",
          stderr,
          "~~~",
        })
        return
      end
      local output = vim.split(result.stdout or "", "\n", { plain = true })
      if #output == 0 or (#output == 1 and output[1] == "") then
        output = { "# Previously on...", "", "No summary output was produced." }
      end
      cached_summary = output
      set_lines(buf, output)
    end)
  end)
end

function M.notify()
  local root = git_root(vim.fn.getcwd())
  if not root or root == "" then
    return
  end

  vim.notify("Previously on summary available. Run :PreviouslyOn or <leader>po.", vim.log.levels.INFO)
end

function M.refresh()
  M.open()
end

vim.api.nvim_create_user_command("PreviouslyOn", function()
  M.open()
end, { desc = "Open the Previously on repository summary" })

vim.api.nvim_create_user_command("PreviouslyOnRefresh", function()
  cached_summary = nil
  M.refresh()
end, { desc = "Refresh the Previously on repository summary" })

if vim.fn.maparg("<leader>po", "n") == "" then
  vim.keymap.set("n", "<leader>po", "<cmd>PreviouslyOn<cr>", { desc = "Previously On summary" })
end

vim.api.nvim_create_autocmd("VimEnter", {
  group = augroup,
  once = true,
  callback = function()
    M.notify()
  end,
})

return M
