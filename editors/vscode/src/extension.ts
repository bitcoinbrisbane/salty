import * as vscode from "vscode";
import { execFile } from "child_process";
import * as path from "path";

export function activate(context: vscode.ExtensionContext): void {
  const disposable = vscode.commands.registerCommand("salty.transpile", () =>
    transpileActiveFile()
  );
  context.subscriptions.push(disposable);
}

export function deactivate(): void {
  // Nothing to clean up.
}

async function transpileActiveFile(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "salty") {
    vscode.window.showErrorMessage("Salty: open a .salty file to transpile.");
    return;
  }

  // Persist unsaved changes so the CLI sees the current source.
  if (editor.document.isDirty) {
    await editor.document.save();
  }

  const filePath = editor.document.uri.fsPath;
  const cli = vscode.workspace
    .getConfiguration("salty")
    .get<string>("cliPath", "salty");

  let solidity: string;
  try {
    solidity = await runSalty(cli, filePath);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`Salty: ${message}`);
    return;
  }

  const doc = await vscode.workspace.openTextDocument({
    language: "solidity",
    content: solidity,
  });
  await vscode.window.showTextDocument(doc, {
    viewColumn: vscode.ViewColumn.Beside,
    preview: true,
  });
}

// runSalty runs `salty transpile <file>` and resolves with its stdout, or
// rejects with the CLI's stderr (e.g. a parse or SIP-2 precision error).
function runSalty(cli: string, filePath: string): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile(
      cli,
      ["transpile", filePath],
      { cwd: path.dirname(filePath) },
      (error, stdout, stderr) => {
        if (error) {
          const detail = stderr.trim() || error.message;
          if ((error as NodeJS.ErrnoException).code === "ENOENT") {
            reject(
              new Error(
                `could not run '${cli}'. Set 'salty.cliPath' or install the CLI (go install ./cmd/salty).`
              )
            );
            return;
          }
          reject(new Error(detail));
          return;
        }
        resolve(stdout);
      }
    );
  });
}
