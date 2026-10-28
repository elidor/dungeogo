"""Two-client walkthrough of the DungeoGo starter-zone milestone.

Starts the real server binary against an empty PostgreSQL database, drives two
telnet-style clients through account and character creation, exploration,
speech, item exchange, disconnect/reconnect and a server restart, and writes
everything each client sent and received, plus a pass/fail check list, to a
Markdown transcript.

Usage: python3 scripts/walkthrough.py BINARY DATABASE_URL OUT.md SERVER.log
The database must be empty; the server listens on 127.0.0.1:47815.
"""
import os
import re
import signal
import socket
import subprocess
import sys
import time

BINARY, DB_URL, OUT, LOG = sys.argv[1:5]
PORT = 47815
ENV = dict(os.environ, PORT=str(PORT), BIND_ADDRESS="127.0.0.1", DATABASE_URL=DB_URL)

transcript = []   # (who, kind, text)
checks = []       # (description, passed)


def note(text):
    transcript.append(("--", "note", text))


def check(description, passed):
    checks.append((description, passed))
    transcript.append(("--", "check", ("PASS " if passed else "FAIL ") + description))


class Server:
    def __init__(self):
        self.log = open(LOG, "a")
        self.proc = None

    def start(self):
        self.proc = subprocess.Popen([BINARY], env=ENV, stdout=self.log, stderr=subprocess.STDOUT, cwd=os.path.dirname(BINARY))
        for _ in range(100):
            try:
                socket.create_connection(("127.0.0.1", PORT), timeout=0.2).close()
                return
            except OSError:
                time.sleep(0.1)
        raise SystemExit("server did not start; see " + LOG)

    def stop(self):
        self.proc.send_signal(signal.SIGTERM)
        return self.proc.wait(timeout=30)


class Client:
    def __init__(self, who):
        self.who = who
        self.sock = socket.create_connection(("127.0.0.1", PORT), timeout=5)
        self.buf = b""

    def expect(self, text, timeout=5):
        deadline = time.time() + timeout
        while text.encode() not in self.buf:
            if time.time() > deadline:
                raise SystemExit(f"{self.who}: timed out waiting for {text!r}; got {self.buf!r}")
            self.sock.settimeout(max(0.05, deadline - time.time()))
            chunk = self.sock.recv(4096)
            if not chunk:
                raise SystemExit(f"{self.who}: connection closed waiting for {text!r}")
            self.buf += chunk
        i = self.buf.index(text.encode()) + len(text)
        got, self.buf = self.buf[:i], self.buf[i:]
        clean = re.sub(rb"\xff[\xfb-\xfe].", b"", got).decode(errors="replace").replace("\r", "")
        transcript.append((self.who, "recv", clean))
        return clean

    def send(self, line, secret=False):
        transcript.append((self.who, "send", "******" if secret else line))
        self.sock.sendall((line + "\r\n").encode())

    def do(self, line, until="\n> "):
        self.send(line)
        return self.expect(until)

    def drain(self, wait=0.3):
        """Collect anything pushed to this client (room messages) without sending."""
        self.sock.settimeout(wait)
        try:
            while True:
                chunk = self.sock.recv(4096)
                if not chunk:
                    break
                self.buf += chunk
        except socket.timeout:
            pass
        got, self.buf = self.buf, b""
        clean = got.decode(errors="replace").replace("\r", "")
        if clean.strip():
            transcript.append((self.who, "recv", clean))
        return clean

    def close(self):
        transcript.append((self.who, "note", "(closes the connection)"))
        self.sock.close()


def create_account(who, name, email, race, klass):
    c = Client(who)
    c.expect("> ")
    c.send(name.lower()); c.expect("Email: ")
    c.send(email); c.expect("(minimum 6 characters):")
    c.send("walkthrough1", secret=True); c.expect("confirm your password")
    c.send("walkthrough1", secret=True)
    out = c.expect("Character> ")
    check(f"{who}: account created", "Account created successfully" in out)
    c.send("create"); c.expect("Name: ")
    c.send(name); c.expect("Race: ")
    c.send(race); c.expect("Class: ")
    c.send(klass); c.expect("Confirm: ")
    c.send("yes")
    out = c.expect("Character> ")
    check(f"{who}: character {name} created", f"Character '{name}' created successfully" in out)
    return c


def enter(c, name):
    c.send("select"); c.expect("Select #: ")
    c.send("1")
    out = c.expect("\n> ")
    check(f"{c.who}: enters the world as {name}", f"Welcome, {name}!" in out)
    return out


def login(who, name):
    c = Client(who)
    c.expect("> ")
    c.send(name.lower()); c.expect("password:")
    c.send("walkthrough1", secret=True)
    out = c.expect("Character> ")
    check(f"{who}: logs back in", "Welcome back" in out)
    return c


server = Server()
note("Server started against an empty database; migrations seed the starter zone.")
server.start()

ana = create_account("Ana's client", "Ana", "ana@example.com", "human", "warrior")
bo = create_account("Bram's client", "Bram", "bram@example.com", "elf", "mage")
out = enter(ana, "Ana")
check("Ana sees the starter items in the starting room", "Training Sword" in out and "Travel Lantern" in out)
enter(bo, "Bram")
ana.drain()

note("Exploration and speech")
out = ana.do("who")
check("who lists both characters", "Ana" in out and "Bram" in out)
ana.do("say Hello, Bram!")
check("Bram hears Ana in the same room", "Ana says: Hello, Bram!" in bo.drain())
bo.do("tell Ana Meet me up north.")
check("Ana receives Bram's tell", "Bram tells you: Meet me up north." in ana.drain())
out = bo.do("north")
check("Bram walks north and sees the Smooth Stone", "Smooth Stone" in out)
check("Ana sees Bram leave", "Bram leaves north." in ana.drain())

note("Items: pick up, give, drop")
out = ana.do("get sword")
check("Ana picks up the Training Sword", "You pick up a Training Sword." in out)
out = ana.do("north")
check("Bram sees Ana arrive", "Ana arrives." in bo.drain())
out = ana.do("give sword Bram")
check("Ana gives the sword to Bram", "You give a Training Sword to Bram." in out)
check("Bram is told he received it", "Ana gives you a Training Sword." in bo.drain())
out = bo.do("inventory")
check("Bram's inventory shows the sword", "Training Sword" in out)
out = ana.do("give sword to Bram")
check("the 'give <item> to <character>' form is refused with usage", "Give what to whom?" in out)
out = bo.do("get stone")
check("Bram picks up the Smooth Stone", "You pick up a Smooth Stone." in out)
check("Ana sees Bram pick it up", "Bram picks up a Smooth Stone." in ana.drain())
out = bo.do("drop stone")
check("Bram drops the stone again", "You drop a Smooth Stone." in out)
ana.drain()

note("Commands outside the milestone say so")
out = bo.do("wear sword")
check("wear is reported as not available", "'wear' isn't available yet" in out)
out = ana.do("kill Bram")
check("kill is reported as not available", "'kill' isn't available yet" in out)
out = ana.do("commands")
check("commands lists what isn't available", "Not available yet:" in out)

note("Disconnect and reconnect")
ana.do("south")
bo.drain()
ana.send("quit")
ana.expect("Saving character")
ana.close()
time.sleep(0.3)
out = bo.do("who")
check("Ana is gone from who after quitting", "Ana" not in out.split("Players currently online:")[-1])
ana = login("Ana's client", "Ana")
out = enter(ana, "Ana")
check("Ana is restored to the starting room", "Travel Lantern" in out)
bo.drain()

note("Server restart: SIGTERM, then start again")
ana.close(); bo.close()
code = server.stop()
check(f"server exits cleanly on SIGTERM (exit code {code})", code == 0)
server.start()
bo = login("Bram's client", "Bram")
out = enter(bo, "Bram")
check("after restart Bram is back in the northern room with the stone on the floor", "Smooth Stone" in out)
out = bo.do("inventory")
check("after restart Bram still carries the Training Sword", "Training Sword" in out)
ana = login("Ana's client", "Ana")
out = enter(ana, "Ana")
check("after restart the starting room no longer has the sword (no respawn)", "Training Sword" not in out and "Travel Lantern" in out)
ana.close(); bo.close()
code = server.stop()
check(f"server exits cleanly again (exit code {code})", code == 0)

with open(OUT, "w") as f:
    passed = sum(1 for _, ok in checks if ok)
    f.write(f"Checks: {passed}/{len(checks)} passed.\n\n")
    for desc, ok in checks:
        f.write(f"- {'PASS' if ok else 'FAIL'}: {desc}\n")
    f.write("\n## Transcript\n\n")
    for who, kind, text in transcript:
        if kind == "note":
            f.write(f"\n### {text}\n\n" if who == "--" else f"_{who} {text}_\n\n")
        elif kind == "check":
            f.write(f"> {text}\n\n")
        else:
            arrow = "→" if kind == "send" else "←"
            body = text.rstrip("\n")
            if kind == "send":
                f.write(f"**{who} {arrow}** `{body}`\n\n")
            else:
                f.write(f"**{who} {arrow}**\n\n```text\n{body}\n```\n\n")
print(f"{sum(ok for _, ok in checks)}/{len(checks)} checks passed")
for desc, ok in checks:
    if not ok:
        print("FAILED:", desc)
sys.exit(0 if all(ok for _, ok in checks) else 1)
