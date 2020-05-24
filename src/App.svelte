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
    let term = null;
    // TODO: Introduce dev/prod switching based on ENV
    var websocket = new WebSocket("wss://progapanda.org/term");
    websocket.binaryType = "arraybuffer"; // ????

    function binaryString(buf) {
      return decodeUTF8(String.fromCharCode.apply(null, new Uint8Array(buf)));
    }

    websocket.onopen = function(evt) {
      term = new Terminal({
        cursorBlink: true,
        screenKeys: true,
        useStyle: true
      });

      if (term) {
        term.setOption("logLevel", "debug");
        const fitAddon = new FitAddon();
        const linksAddon = new WebLinksAddon();
        term.loadAddon(fitAddon);
        term.loadAddon(linksAddon);

        term.onData(function(data) {
          websocket.send(new TextEncoder().encode("\x00" + data));
        });

        term.onResize(function(evt) {
          console.log(term.rows, "x", term.cols);
          websocket.send(
            new TextEncoder().encode(
              "\x01" + JSON.stringify({ cols: evt.cols, rows: evt.rows })
            )
          );
        });

        websocket.onmessage = function(evt) {
          term.write(binaryString(evt.data));
          fitAddon.fit();
        };

        websocket.onclose = function(evt) {
          if (term) {
            term.write("Session terminated");
          }
        };

        websocket.onerror = function(evt) {
          if (typeof console.log == "function") {
            console.log(evt);
          }
        };

        term.onTitleChange(function(title) {
          document.title = title;
        });

        term.setOption("fontFamily", "VT323-Regular");
        term.setOption("fontSize", 24);

        term.open(terminalDiv);
        term.focus();

        window.addEventListener("resize", () => {
          fitAddon.fit();
        });
      }
    };
  });
</script>

<style>
  :global(body) {
    margin: 0;
    background-color: black;
  }

  main {
    background-color: black;
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
