# Lumpfish Media Catcher

A simple, user-friendly application that allows you to download videos and audio from many popular websites and [many more](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md).

This is just a graphical frontend for the command-line application [yt-dlp](https://github.com/yt-dlp/yt-dlp), which is where the magic happens. With this project, non-technical users can take advantage of the comprehensive list of supported websites and download features of yt-dlp.

## How to install and use it

1. Download pre-built binaries from the [Releases](https://github.com/lumpfishtech/mediacatcher/releases) page:

- **Windows (64-bit)**: Download the `.zip` file, extract, and run `LumpfishMediaCatcher.exe`
- **Linux (64-bit)**: Download the `.tar.gz` file, extract, and run `./LumpfishMediaCatcher`

2. Extract the executable and place it in an empty folder (a config file and log file will be created on the first run)
3. Double-click on the executable
4. On first run, you will be presented with a setup wizard. Choose to use the system yt-dlp if you have already installed it, or let Lumpfish Media Catcher manage it.
5. Find the URL of the video you want to download
6. In the main application window, paste the URL in the field and click "List Formats". This will populate the dropdown menu below where you can pick the desired format (if a format does not specify "audio only" or "video only", it contains both tracks).
7. Change the download folder or accept the default, then click "Download"


## Technical details

### Building from Source

There is a simple Makefile to build, test and run the application:

```bash
git clone https://github.com/lumpfishtech/mediacatcher.git
cd mediacatcher
make build
make test
make run
make clean
```

### Logging

The application creates a log file (`lmc-app.log`) in the same directory as the executable. This makes the application portable - when you move the executable to a new location, logs will be created in that new location.

Edit the `log_file` field in `lmc-config.json` to change the log name or location (e.g., `"my-log.log"`, `"../logs/app.log"`, or `"/var/log/app.log"`). Relative or absolute paths work if the folder structure already exists.

## About This Project

This project is an experiment in AI-driven software development. I developed this application almost entirely using Claude AI (Anthropic's AI assistant) to explore how effectively an AI agent can assist in real-world software development.

The development process involved using prompt engineering - giving Claude incremental instructions like:
- "Create the basic structure of the application"
- "Add this feature"
- "Fix this problem I found while using the application"

### Key Learnings

Through this experience, I found that the most effective approach is:

1. **Incremental development**: Break the project into small, manageable tasks rather than asking for large features all at once
2. **Regular code reviews**: Periodically ask the AI to review the codebase to identify and reduce code duplication, potential bugs, and architectural issues
3. **Collaborative decision-making**: Discuss implementation details and architectural decisions with the AI rather than just accepting the first solution
4. **Proper guidance**: The quality of results depends heavily on how well you guide the AI agent - clear, specific prompts lead to better outcomes

I may expand this section in the future with more detailed insights about the development process and results.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
