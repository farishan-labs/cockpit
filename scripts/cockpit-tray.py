#!/usr/bin/env python3
# Ubuntu tray indicator for the FHL cockpit daemon.
import os
import subprocess
import webbrowser

import gi
gi.require_version("Gtk", "3.0")
gi.require_version("AyatanaAppIndicator3", "0.1")
from gi.repository import Gtk, GLib, AyatanaAppIndicator3

HOME = os.path.expanduser("~")
COCKPIT = os.path.join(HOME, ".local", "bin", "cockpit")
URL = "http://127.0.0.1:7777"

def is_running():
    try:
        out = subprocess.check_output([COCKPIT, "status"], text=True, stderr=subprocess.DEVNULL)
        return "running" in out
    except Exception:
        return False

class Tray:
    def __init__(self):
        self.ind = AyatanaAppIndicator3.Indicator.new(
            "cockpit", "utilities-terminal", AyatanaAppIndicator3.IndicatorCategory.APPLICATION_STATUS)
        self.ind.set_status(AyatanaAppIndicator3.IndicatorStatus.ACTIVE)
        self.ind.set_menu(self.build_menu())
        GLib.timeout_add_seconds(5, self.refresh)

    def build_menu(self):
        menu = Gtk.Menu()
        open_item = Gtk.MenuItem(label="Open Cockpit UI")
        open_item.connect("activate", lambda _: webbrowser.open(URL))
        menu.append(open_item)

        start_item = Gtk.MenuItem(label="Start Cockpit")
        start_item.connect("activate", lambda _: subprocess.Popen([COCKPIT], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL))
        menu.append(start_item)

        stop_item = Gtk.MenuItem(label="Stop Cockpit")
        stop_item.connect("activate", lambda _: subprocess.run([COCKPIT, "stop"]))
        menu.append(stop_item)

        menu.append(Gtk.SeparatorMenuItem())

        quit_item = Gtk.MenuItem(label="Quit Tray")
        quit_item.connect("activate", lambda _: Gtk.main_quit())
        menu.append(quit_item)

        menu.show_all()
        return menu

    def refresh(self):
        if is_running():
            self.ind.set_icon_full("utilities-terminal", "cockpit running")
        else:
            self.ind.set_icon_full("dialog-warning", "cockpit stopped")
        return True

if __name__ == "__main__":
    Tray()
    Gtk.main()
