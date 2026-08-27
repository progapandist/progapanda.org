import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import "./main.css";

const terminalElement = document.querySelector("#xterm");

function openTrustedLink(_event, uri) {
  try {
    const url = new URL(uri);
    if (url.protocol !== "https:" || url.hostname !== "github.com") {
      return;
    }

    window.open(url.href, "_blank", "noopener,noreferrer");
  } catch {
    // Ignore malformed terminal-provided links.
  }
}

const terminal = new Terminal({
  cursorBlink: true,
  fontFamily:
    'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
  fontSize: 17,
  lineHeight: 1,
  linkHandler: {
    activate: openTrustedLink,
  },
  theme: {
    background: "#0b0b0f",
    foreground: "#eeeeee",
    cursor: "#ff87d7",
    cursorAccent: "#0b0b0f",
    selectionBackground: "rgba(255, 135, 215, 0.3)",
    black: "#0b0b0f",
    brightBlack: "#626262",
    magenta: "#d75faf",
    brightMagenta: "#ff87d7",
    cyan: "#5fafd7",
    brightCyan: "#5fd7ff",
    white: "#d0d0d0",
    brightWhite: "#eeeeee",
  },
});
const fitAddon = new FitAddon();

terminal.loadAddon(fitAddon);
terminal.loadAddon(new WebLinksAddon());
terminal.open(terminalElement);
terminal.focus();

const encoder = new TextEncoder();
const promptDecoder = new TextDecoder();
const websocketProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
let websocket = null;
let disposed = false;
let receivedOutput = false;
let commandPrefilled = false;
let outputTail = "";

const spinnerFrames = ["◒", "◐", "◓", "◑"];
let spinnerFrame = 0;

function drawLoading() {
  terminal.write(
    `\r\x1b[2K\x1b[38;5;212m${spinnerFrames[spinnerFrame]}\x1b[0m ` +
      "\x1b[38;5;81mWAKING THE PANDA\x1b[0m " +
      "\x1b[38;5;245m· preparing your private terminal\x1b[0m",
  );
  spinnerFrame = (spinnerFrame + 1) % spinnerFrames.length;
}

drawLoading();
const spinner = window.setInterval(drawLoading, 120);

function send(type, payload) {
  if (!websocket || websocket.readyState !== WebSocket.OPEN) {
    return;
  }

  const body = encoder.encode(payload);
  const message = new Uint8Array(body.length + 1);
  message[0] = type;
  message.set(body, 1);
  websocket.send(message);
}

function sendSize({ cols = terminal.cols, rows = terminal.rows } = {}) {
  send(1, JSON.stringify({ cols, rows }));
}

terminal.onData((data) => send(0, data));
terminal.onResize(sendSize);
terminal.onTitleChange((title) => {
  document.title = title;
});

function connect() {
  if (disposed) {
    return;
  }

  fitAddon.fit();
  websocket = new WebSocket(
    `${websocketProtocol}//${window.location.host}/term`,
  );
  websocket.binaryType = "arraybuffer";

  websocket.addEventListener("open", () => {
    fitAddon.fit();
    sendSize();
  });

  websocket.addEventListener("message", (event) => {
    const output = new Uint8Array(event.data);
    if (!receivedOutput) {
      receivedOutput = true;
      window.clearInterval(spinner);
      terminal.write("\x1b[2J\x1b[H");
    }
    terminal.write(output);

    outputTail = (
      outputTail + promptDecoder.decode(output, { stream: true })
    ).slice(-256);
    if (!commandPrefilled && outputTail.includes("/app $ ")) {
      commandPrefilled = true;
      send(0, "./hello2");
    }
  });

  websocket.addEventListener("close", () => {
    window.clearInterval(spinner);
    if (!disposed) {
      terminal.write("\r\n\x1b[38;5;245mSession terminated\x1b[0m");
    }
  });

  websocket.addEventListener("error", (event) => {
    console.error("Terminal WebSocket error", event);
  });
}

const resizeObserver = new ResizeObserver(() => fitAddon.fit());
resizeObserver.observe(terminalElement);

// Let the browser establish final container geometry before the server creates
// the PTY. The first WebSocket message carries these settled dimensions.
window.requestAnimationFrame(() => {
  window.requestAnimationFrame(connect);
});

window.addEventListener("beforeunload", () => {
  disposed = true;
  window.clearInterval(spinner);
  resizeObserver.disconnect();
  if (websocket) {
    websocket.close();
  }
  terminal.dispose();
});
