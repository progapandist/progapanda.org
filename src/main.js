import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import "./main.css";

const terminalElement = document.querySelector("#xterm");

// A coarse pointer means no keyboard to press Enter with, and fingers instead
// of a wheel. Both of those need handling below.
const touchOnly = window.matchMedia("(pointer: coarse)").matches;

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

// A full-screen app (the TUI) owns the alternate buffer and asks for mouse
// reporting; the shell does neither. That difference decides who handles
// scrolling and whether a keyboard is any use.
const fullscreenApp = () => terminal.buffer.active.type === "alternate";

// Xterm reads keystrokes through a hidden textarea, so tapping the terminal
// focuses it and iOS raises the keyboard over the UI. A full-screen app is
// driven by taps and swipes and has nothing to type into, so suppress the
// keyboard while the alternate buffer is active — and hand it back the moment
// the app exits and the shell prompt returns.
function syncKeyboard() {
  const input = terminal.textarea;
  if (!input) {
    return;
  }

  const fullscreen = fullscreenApp();
  input.inputMode = fullscreen ? "none" : "text";

  // iOS only offers the keyboard when a blurred element gains focus, and
  // leaving the app does not move focus. Let go of it, so the next tap counts
  // as a fresh focus. On desktop keep it, or typing would stop working after
  // quitting the TUI.
  if (touchOnly && !fullscreen) {
    input.blur();
  }
}

terminal.buffer.onBufferChange(syncKeyboard);
syncKeyboard();

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
  // The container cannot tell a phone from a narrow window, and one of the
  // programs in there wants to know. Device capability, not anything personal.
  websocket = new WebSocket(
    `${websocketProtocol}//${window.location.host}/term${
      touchOnly ? "?pointer=coarse" : ""
    }`,
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
      send(0, "hello2");
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

const WHEEL_UP = 64;
const WHEEL_DOWN = 65;
// Each wheel event scrolls three lines in the TUI's viewport.
const LINES_PER_EVENT = 3;

const rowHeight = () => terminalElement.clientHeight / terminal.rows || 20;

// Every wheel event costs a round trip and a repaint inside a 0.1-CPU
// container — about 45ms. Wheels and touchmove both fire far faster than that,
// and sending each one separately queues repaints the server cannot keep up
// with, which reads as lag. So accumulate and send at most one message per
// FLUSH_MS, carrying everything since the last: the TUI applies them all and
// repaints once.
const FLUSH_MS = 40;
const MAX_PER_FLUSH = 6;
let pendingWheel = 0;
let wheelCarry = 0;
let flushTimer = null;
let lastFlush = 0;

function flushWheel() {
  flushTimer = null;
  lastFlush = performance.now();
  if (pendingWheel === 0) {
    return;
  }

  const button = pendingWheel > 0 ? WHEEL_DOWN : WHEEL_UP;
  const count = Math.min(Math.abs(pendingWheel), MAX_PER_FLUSH);
  pendingWheel = 0;
  send(0, `\x1b[<${button};1;1M`.repeat(count));
}

function queueWheel(events) {
  pendingWheel += events;
  if (flushTimer !== null) {
    return;
  }
  flushTimer = window.setTimeout(
    flushWheel,
    Math.max(0, FLUSH_MS - (performance.now() - lastFlush)),
  );
}

// Take the wheel away from Xterm so it goes through the same batching. Left to
// itself it emits one sequence per tick, which is what made a trackpad flick
// crawl on the desktop too.
terminal.attachCustomWheelEventHandler((event) => {
  // At the shell, Xterm scrolls its own scrollback and nothing is listening
  // for mouse reports — sending them there would type escape sequences at the
  // prompt.
  if (!fullscreenApp()) {
    return true;
  }

  // Carry the fraction over between events. A trackpad sends a stream of
  // small deltas, and truncating each one on its own rounds every last one of
  // them to zero — which, since we also stop Xterm handling it, means nothing
  // scrolls at all.
  const perEvent =
    event.deltaMode === 1 ? LINES_PER_EVENT : rowHeight() * LINES_PER_EVENT;
  wheelCarry += event.deltaY / perEvent;
  const events = Math.trunc(wheelCarry);
  if (events !== 0) {
    wheelCarry -= events;
    queueWheel(events);
  }
  return false;
});

// The TUI asks Xterm to report mouse events to it, which leaves touch gestures
// meaning nothing at all. Translate a vertical swipe into the wheel events its
// viewport already understands. Taps are left alone so they still select a
// menu item.
if (touchOnly) {
  let anchorY = null;

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
      if (anchorY === null || event.touches.length !== 1 || !fullscreenApp()) {
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

      queueWheel(events);
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
