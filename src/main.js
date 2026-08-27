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

// A coarse pointer means no keyboard to press Enter with, and fingers instead
// of a wheel. Both of those need handling below.
const touchOnly = window.matchMedia("(pointer: coarse)").matches;

const spinnerFrames = ["◒", "◐", "◓", "◑"];
let spinnerFrame = 0;

function drawLoading() {
  const frame = spinnerFrames[spinnerFrame];
  let message = `\x1b[38;5;212m${frame}\x1b[0m`;
  if (terminal.cols >= 19) {
    message += " \x1b[38;5;81mWAKING THE PANDA\x1b[0m";
  }
  if (terminal.cols >= 53) {
    message +=
      " \x1b[38;5;245m· preparing your private terminal\x1b[0m";
  }

  // A resize can reflow the previous frame across several rows. Clearing the
  // display, rather than only the cursor row, prevents those rows accumulating
  // on narrow mobile terminals.
  terminal.write(`\x1b[2J\x1b[H${message}`);
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
      // "Press Enter to continue" is not an option without a keyboard, so run
      // it after a beat long enough to read the banner.
      if (touchOnly) {
        window.setTimeout(() => send(0, "\r"), 1200);
      }
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

// The TUI asks Xterm to report mouse events to it, which leaves touch gestures
// meaning nothing at all. Translate a vertical swipe into the wheel events its
// viewport already understands. Taps are left alone so they still select a
// menu item.
if (touchOnly) {
  const WHEEL_UP = 64;
  const WHEEL_DOWN = 65;
  // Each wheel event scrolls three lines, so require three rows of travel per
  // event and the content keeps pace with the finger.
  const LINES_PER_EVENT = 3;
  let anchorY = null;

  const rowHeight = () =>
    terminalElement.clientHeight / terminal.rows || 20;

  terminalElement.addEventListener(
    "touchstart",
    (event) => {
      anchorY = event.touches.length === 1 ? event.touches[0].clientY : null;
    },
    { passive: true },
  );

  terminalElement.addEventListener(
    "touchmove",
    (event) => {
      if (anchorY === null || event.touches.length !== 1) {
        return;
      }

      const step = rowHeight() * LINES_PER_EVENT;
      const travelled = anchorY - event.touches[0].clientY;
      const events = Math.trunc(travelled / step);
      if (events === 0) {
        return;
      }

      // Only now is this a scroll rather than a tap, so only now claim the
      // gesture from the browser's own panning.
      event.preventDefault();
      anchorY -= events * step;

      const button = events > 0 ? WHEEL_DOWN : WHEEL_UP;
      for (let i = 0; i < Math.min(Math.abs(events), 8); i += 1) {
        send(0, `\x1b[<${button};1;1M`);
      }
    },
    { passive: false },
  );

  terminalElement.addEventListener(
    "touchend",
    () => {
      anchorY = null;
    },
    { passive: true },
  );
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
