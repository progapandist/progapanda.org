import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import "./font.css";

const terminalElement = document.querySelector("#xterm");
const terminal = new Terminal({
  cursorBlink: true,
  fontFamily: "VT323-Regular",
  fontSize: 24,
});
const fitAddon = new FitAddon();

terminal.loadAddon(fitAddon);
terminal.loadAddon(new WebLinksAddon());
terminal.open(terminalElement);
terminal.focus();

const websocketProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
const websocket = new WebSocket(
  `${websocketProtocol}//${window.location.host}/term`,
);
websocket.binaryType = "arraybuffer";

const encoder = new TextEncoder();

function send(type, payload) {
  if (websocket.readyState !== WebSocket.OPEN) {
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

websocket.addEventListener("open", () => {
  fitAddon.fit();
  sendSize();
});

websocket.addEventListener("message", (event) => {
  // Xterm owns the stateful UTF-8 decoder, so a character may safely span
  // multiple WebSocket messages.
  terminal.write(new Uint8Array(event.data));
});

websocket.addEventListener("close", () => {
  terminal.write("\r\nSession terminated");
});

websocket.addEventListener("error", (event) => {
  console.error("Terminal WebSocket error", event);
});

terminal.onTitleChange((title) => {
  document.title = title;
});

const resizeObserver = new ResizeObserver(() => fitAddon.fit());
resizeObserver.observe(terminalElement);

window.addEventListener("beforeunload", () => {
  resizeObserver.disconnect();
  websocket.close();
  terminal.dispose();
});
