# tinyMem Quick Start Guide (for Beginners)

Welcome to tinyMem! This guide will help you get up and running, even if you're not a command-line expert.

## What is tinyMem?

Think of tinyMem as a short-term memory for your AI assistant. It runs quietly in the background on your computer and helps your AI remember the context of your conversation during a coding session. This means you don't have to repeat yourself, and the AI can give you more relevant and accurate answers.

## Part 1: Installation

First, you need to download the correct file for your operating system.

### For Windows Users

1.  **Go to the Releases Page:** Open your web browser and go to [https://github.com/andrzejmarczewski/tinyMem/releases](https://github.com/andrzejmarczewski/tinyMem/releases).
2.  **Download tinyMem:** Look for a file named `tinymem-windows-amd64.exe` and click on it to download.
3.  **Create a Folder:** On your Desktop, create a new folder and name it `tinyMem`.
4.  **Move the File:** Move the downloaded `tinymem-windows-amd64.exe` file into the `tinyMem` folder you just created.
5.  **Rename (Recommended):** For simplicity, rename the file from `tinymem-windows-amd64.exe` to `tinymem.exe`.

### For macOS Users

1.  **Go to the Releases Page:** Open your web browser and go to [https://github.com/andrzejmarczewski/tinyMem/releases](https://github.com/andrzejmarczewski/tinyMem/releases).
2.  **Download tinyMem:**
    *   If you have a Mac with an Apple chip (M1, M2, M3, etc.), download `tinymem-darwin-arm64`.
    *   If you have a Mac with an Intel chip, download `tinymem-darwin-amd64`.
3.  **Create a Folder:** On your Desktop, create a new folder and name it `tinyMem`.
4.  **Move the File:** Move the downloaded file into the `tinyMem` folder.
5.  **Rename (Recommended):** For simplicity, rename the file to `tinymem`.

## Part 2: Running tinyMem

Now that you have the file, you need to run it.

### For Windows Users

1.  **Open the tinyMem Folder:** Double-click the `tinyMem` folder on your Desktop.
2.  **Open Command Prompt:** In the address bar at the top of the File Explorer window, type `cmd` and press Enter.
    ![Windows CMD in Address Bar](https://i.imgur.com/8h2Z9sK.png)
3.  **Run tinyMem:** A black window (the Command Prompt) will appear. In this window, type `tinymem.exe proxy` and press Enter.

### For macOS Users

1.  **Open the Terminal App:** You can find it in your `Applications` folder, inside the `Utilities` subfolder. You can also search for "Terminal" in Spotlight (the magnifying glass icon in the top-right corner of your screen).
2.  **Navigate to the Folder:** In the Terminal window, type `cd ~/Desktop/tinyMem` and press Enter. This command changes the directory to the `tinyMem` folder on your Desktop.
3.  **Make it Executable:** Type `chmod +x tinymem` and press Enter. This gives your computer permission to run the file.
4.  **Run tinyMem:** Now, type `./tinymem proxy` and press Enter.

**macOS Security Warning:** The first time you run it, you might see a security warning saying the developer cannot be verified. If this happens:
1.  Click "Cancel" on the warning popup.
2.  Go to "System Settings" > "Privacy & Security".
3.  Scroll down, and you will see a message about "tinymem" being blocked. Click the "Allow Anyway" or "Open Anyway" button.
4.  Go back to the Terminal and run the `./tinymem proxy` command again.

## Part 3: What Now?

You should see some text in the terminal, and the cursor will be on a new line, but it might not be blinking. This is good! It means `tinyMem` is running and listening for your AI assistant.

**Leave this terminal window open!** As long as it's running, `tinyMem` is working its magic in the background. If you close the window, `tinyMem` will stop.

The final step is to tell your AI assistant to use `tinyMem`. This process is different for each AI tool (like VS Code with Continue, Cursor, etc.).

For detailed instructions on how to connect your specific tool, please refer to the main `README.md` file's [IDE Integration](httpsDE_Integration) section.
