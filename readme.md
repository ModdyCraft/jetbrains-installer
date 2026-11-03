# 🚀 JetBrains Linux Installer

Hello! 👋 Welcome to this project.

It was born on a free afternoon of experimentation working hand in hand with **Gemini**, with the idea of simplifying the installation and uninstallation of JetBrains IDEs on Linux distributions quickly and hassle-free directly from the terminal.

## 📌 About the project

I currently maintain this tool as a hobby to improve workflow in Linux environments.

I know this project probably has an "expiration date" or a natural end, as **JetBrains** will eventually offer more advanced official installers or integrations for Linux. However, in the meantime, I hope to keep improving the tool, polishing details, and adding useful features.

---

## 🗺️ Roadmap / Upcoming improvements

* [ ] **Multi-language support**: Translations to other languages (such as English) are right around the corner.
* [ ] **Graphical User Interface (GUI)**: I plan to add an accessible visual interface in the not-too-distant future for those who prefer not to use strictly the terminal.

---

## 🐧 Requirements & Installation

> ⚠️ **Important**: This executable is designed **exclusively for Linux systems**.

### 🛠️ Building

If you want to compile the project yourself from source, simply run the build script:

```bash
./build.sh
```

or

```bash
go build -ldflags="-s -w" -o build/jetbrains-installer .
```

---

### 🚀 Execution

#### 1. Grant execution permissions

If you download the binary directly or after compiling it, make sure to grant it execution permissions in the terminal:

```bash
chmod +x jetbrains-installer

```

#### 2. Run the tool

Since the tool manages shortcuts in `/usr/share/applications` and moves folders inside `/opt/jetbrains`, it must be run with superuser privileges:

```bash
sudo ./jetbrains-installer

```

---

## 📜 License

This project is distributed under the open-source **MIT License**. See the [`LICENSE`](./LICENSE) file for more details.

---

*Thanks for checking out the project! If you have any suggestions or encounter any issues, feel free to open an issue or pull request.*