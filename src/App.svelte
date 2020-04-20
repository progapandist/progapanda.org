<script>
  import { Terminal } from "xterm";
  import { onMount } from "svelte";
  import { FitAddon } from "xterm-addon-fit";
  import { WebLinksAddon } from "xterm-addon-web-links";
  import "xterm/css/xterm.css";

  let UTF8Decoder = function() {
    // The number of bytes left in the current sequence.
    this.bytesLeft = 0;
    // The in-progress code point being decoded, if bytesLeft > 0.
    this.codePoint = 0;
    // The lower bound on the final code point, if bytesLeft > 0.
    this.lowerBound = 0;
  };

  /**
   * Decodes a some UTF-8 data, taking into account state from previous
   * data streamed through the encoder.
   *
   * @param {String} str data to decode, represented as a JavaScript
   *     String with each code unit representing a byte between 0x00 to
   *     0xFF.
   * @return {String} The data decoded into a JavaScript UTF-16 string.
   */
  UTF8Decoder.prototype.decode = function(str) {
    var ret = "";
    for (var i = 0; i < str.length; i++) {
      var c = str.charCodeAt(i);
      if (this.bytesLeft == 0) {
        if (c <= 0x7f) {
          ret += str.charAt(i);
        } else if (0xc0 <= c && c <= 0xdf) {
          this.codePoint = c - 0xc0;
          this.bytesLeft = 1;
          this.lowerBound = 0x80;
        } else if (0xe0 <= c && c <= 0xef) {
          this.codePoint = c - 0xe0;
          this.bytesLeft = 2;
          this.lowerBound = 0x800;
        } else if (0xf0 <= c && c <= 0xf7) {
          this.codePoint = c - 0xf0;
          this.bytesLeft = 3;
          this.lowerBound = 0x10000;
        } else if (0xf8 <= c && c <= 0xfb) {
          this.codePoint = c - 0xf8;
          this.bytesLeft = 4;
          this.lowerBound = 0x200000;
        } else if (0xfc <= c && c <= 0xfd) {
          this.codePoint = c - 0xfc;
          this.bytesLeft = 5;
          this.lowerBound = 0x4000000;
        } else {
          ret += "\ufffd";
        }
      } else {
        if (0x80 <= c && c <= 0xbf) {
          this.bytesLeft--;
          this.codePoint = (this.codePoint << 6) + (c - 0x80);
          if (this.bytesLeft == 0) {
            // Got a full sequence. Check if it's within bounds and
            // filter out surrogate pairs.
            var codePoint = this.codePoint;
            if (
              codePoint < this.lowerBound ||
              (0xd800 <= codePoint && codePoint <= 0xdfff) ||
              codePoint > 0x10ffff
            ) {
              ret += "\ufffd";
            } else {
              // Encode as UTF-16 in the output.
              if (codePoint < 0x10000) {
                ret += String.fromCharCode(codePoint);
              } else {
                // Surrogate pair.
                codePoint -= 0x10000;
                ret += String.fromCharCode(
                  0xd800 + ((codePoint >>> 10) & 0x3ff),
                  0xdc00 + (codePoint & 0x3ff)
                );
              }
            }
          }
        } else {
          // Too few bytes in multi-byte sequence. Rewind stream so we
          // don't lose the next byte.
          ret += "\ufffd";
          this.bytesLeft = 0;
          i--;
        }
      }
    }
    return ret;
  };

  let decodeUTF8 = function(utf8) {
    return new UTF8Decoder().decode(utf8);
  };

  let terminalDiv;

  onMount(() => {
    let term = null;
    var websocket = new WebSocket("ws://localhost:4567/term");
    websocket.binaryType = "arraybuffer"; // ????

    function ab2str(buf) {
      return decodeUTF8(String.fromCharCode.apply(null, new Uint8Array(buf)));
    }

    websocket.onopen = function(evt) {
      term = new Terminal({
        cursorBlink: true,
        screenKeys: true,
        useStyle: true
      });

      if (term) {
        const fitAddon = new FitAddon();
        const linksAddon = new WebLinksAddon();
        term.loadAddon(fitAddon);
        term.loadAddon(linksAddon);
        term.setOption("logLevel", "debug");

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

        term.onTitleChange(function(title) {
          document.title = title;
        });

        term.open(terminalDiv);
        fitAddon.fit();
        term.focus();
        linksAddon.activate();
      }
    };

    websocket.onmessage = function(evt) {
      term.write(ab2str(evt.data));
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
  });
</script>

<style>
  :global(body) {
    /* this will apply to <body> */
    margin: 0;
    background-color: black;
  }

  main {
    background-color: black;
    margin: 0;
    height: 80vh;
  }

  #xterm {
    width: 100%;
    height: 100%;
  }
</style>

<main>
  <div bind:this={terminalDiv} id="xterm" />
</main>
