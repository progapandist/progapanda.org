<script>
  import "./font.css";
  import { Terminal } from "xterm";
  import { onMount } from "svelte";
  import { FitAddon } from "xterm-addon-fit";
  import { WebLinksAddon } from "xterm-addon-web-links";
  import decodeUTF8 from "./decoder";
  import "xterm/css/xterm.css";

  let terminalDiv;

  onMount(() => {
    const term = new Terminal({
      cursorBlink: true,
      screenKeys: true,
      useStyle: true,
      fontFamily: "VT323-Regular",
      fontSize: 24,
      theme: {
        background: "#0b0b0f",
        foreground: "#eeeeee",
        cursor: "#ff87d7",
        cursorAccent: "#0b0b0f",
        selection: "rgba(255, 135, 215, 0.3)",
        black: "#0b0b0f",
        brightBlack: "#626262",
        magenta: "#d75faf",
        brightMagenta: "#ff87d7",
        cyan: "#5fafd7",
        brightCyan: "#5fd7ff",
        white: "#d0d0d0",
        brightWhite: "#eeeeee"
      }
    });
    const fitAddon = new FitAddon();
    const linksAddon = new WebLinksAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(linksAddon);
    term.open(terminalDiv);
    fitAddon.fit();
    term.focus();

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const websocket = new WebSocket(`${protocol}//${window.location.host}/term`);
    websocket.binaryType = "arraybuffer";
    let receivedOutput = false;
    let commandPrefilled = false;
    let outputTail = "";
    const spinnerFrames = ["◒", "◐", "◓", "◑"];
    let spinnerFrame = 0;

    function drawLoading() {
      term.write(
        `\r\x1b[2K\x1b[38;5;212m${spinnerFrames[spinnerFrame]}\x1b[0m ` +
        "\x1b[38;5;81mWAKING THE PANDA\x1b[0m " +
        "\x1b[38;5;245m· preparing your private terminal\x1b[0m"
      );
      spinnerFrame = (spinnerFrame + 1) % spinnerFrames.length;
    }
    drawLoading();
    const spinner = window.setInterval(drawLoading, 120);

    function binaryString(buf) {
      return decodeUTF8(String.fromCharCode.apply(null, new Uint8Array(buf)));
    }

    websocket.onopen = function(evt) {
      term.onData(function(data) {
        websocket.send(new TextEncoder().encode("\x00" + data));
      });

      term.onResize(function(evt) {
        websocket.send(
          new TextEncoder().encode(
            "\x01" + JSON.stringify({ cols: evt.cols, rows: evt.rows })
          )
        );
      });

      websocket.send(
        new TextEncoder().encode(
          "\x01" + JSON.stringify({ cols: term.cols, rows: term.rows })
        )
      );
    };

    websocket.onmessage = function(evt) {
      const output = binaryString(evt.data);
      if (!receivedOutput) {
        receivedOutput = true;
        window.clearInterval(spinner);
        term.write("\x1b[2J\x1b[H");
      }
      term.write(output);

      // Put actual bytes into the guest shell, rather than merely painting
      // text in Xterm. The command remains editable and Enter launches it.
      outputTail = (outputTail + output).slice(-256);
      if (!commandPrefilled && outputTail.includes("/app $ ")) {
        commandPrefilled = true;
        websocket.send(new TextEncoder().encode("\x00./hello2"));
      }
    };

    websocket.onclose = function(evt) {
      window.clearInterval(spinner);
      term.write("\r\n\x1b[38;5;245mSession terminated\x1b[0m");
    };

    websocket.onerror = function(evt) {
      if (typeof console.log == "function") {
        console.log(evt);
      }
    };

    term.onTitleChange(function(title) {
      document.title = title;
    });

    const resize = () => fitAddon.fit();
    window.addEventListener("resize", resize);

    return () => {
      window.clearInterval(spinner);
      window.removeEventListener("resize", resize);
      websocket.close();
      term.dispose();
    };
  });
</script>

<style>
  :global(body) {
    margin: 0;
    background-color: #0b0b0f;
  }

  main {
    background-color: #0b0b0f;
    margin: 0;
    height: 100vh;
  }

  #xterm {
    width: 100%;
    height: 100%;
  }
</style>

<main>
  <div bind:this={terminalDiv} id="xterm" />
</main>
