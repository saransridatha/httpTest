# httpTest: A Powerful Web Traffic Simulator

`httptest` is a simple yet powerful command-line tool designed for simulating web traffic. It's an invaluable asset for stress testing your web applications, benchmarking API performance, and understanding how your infrastructure handles concurrent requests under various conditions.

## ⚠️ Ethical Use Disclaimer

`httptest` is a tool designed for developers and DevOps engineers to perform load testing on applications that they own or have explicit permission to test.

Using this tool to test third-party servers or websites without prior written consent is strictly prohibited. Unauthorized testing can be mistaken for a denial-of-service (DoS) attack. Users are solely responsible for their actions and must comply with all applicable laws. The author assumes no liability for any misuse of this tool.
## Features

`httptest` provides a robust set of features to help you conduct effective load tests:

*   **Easy to Use**: Configure your tests effortlessly with intuitive command-line flags.
*   **Concurrent Requests**: Simulate realistic load by sending a large number of requests concurrently.
*   **Multiple HTTP Methods**: Supports all standard HTTP methods, including `GET`, `POST`, `PUT`, `DELETE`, and more.
*   **Custom Payloads**: Attach request bodies directly from the command line or load them from a file for complex scenarios.
*   **Custom Headers**: Include custom HTTP headers to mimic specific client behaviors or authentication flows.
*   **Flexible Test Modes**: Run tests based on a fixed total number of requests or for a specified duration.
*   **Detailed & Colorful Summary**: Get a comprehensive, easy-to-read summary of your test results with color-coded output for quick insights.
*   **Response Time Histogram**: Visualize the distribution of response times to quickly identify performance bottlenecks and outliers.
*   **JSON Output**: Export the complete summary report to a JSON file for further analysis and integration with other tools.

## Installation

Getting `httptest` up and running on your system is straightforward.

### Prerequisites

Ensure you have Go (version 1.22 or higher recommended) installed on your system. You can download it from [go.dev/doc/install](https://go.dev/doc/install).

### Automated Installation (Recommended)

The easiest way to install or update `httptest` is by using the provided `install.sh` script. This script handles everything from cloning the repository to configuring your system's `PATH`.

```bash
git clone https://github.com/saransridatha/httptest.git && cd httptest && bash install.sh && echo "Installation complete! Please run 'source ~/.bashrc' (or '~/.zshrc') or open a new terminal to use 'httptest'."
```

**What this command does:**

*   `git clone ...`: Downloads the `httptest` project to your local machine. If the directory already exists, it will skip cloning.
*   `cd ...`: Navigates into the newly cloned project directory.
*   `bash install.sh`: Executes the setup script, which performs the following actions:
    *   Verifies that Go is installed on your system.
    *   Installs or updates the `httptest` tool using `go install`.
    *   Automatically configures your shell's `PATH` environment variable (for Bash or Zsh users) to ensure `httptest` is accessible from any directory.
    *   Cleans up the cloned repository directory after a successful installation.
*   `echo ...`: Displays a final message with important instructions for completing the setup.

**Important Final Step:**

After running the automated installation command, you *must* reload your shell to apply the `PATH` changes. You can do this by running:

```bash
source ~/.bashrc
```

*(Note: If you use a different shell like Zsh, you might need to run `source ~/.zshrc` instead.)*

Once your shell is reloaded, you can verify the installation by checking the tool's help message:

```bash
httptest -h
```

### Manual Installation

If you prefer to install `httptest` manually, follow these steps:

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/saransridatha/httptest.git
    cd httptest
    ```
2.  **Build the executable:**
    ```bash
    go build script.go
    ```
3.  **Move the executable to your PATH:**
    ```bash
    sudo mv httptest /usr/local/bin/ # Or any other directory included in your system's PATH
    ```

## Usage

`httptest` is designed to be flexible and easy to use. Here are some common usage examples:

### 1. Simple GET Request Load Test

Send 1000 GET requests to `https://example.com` with a concurrency of 100:

```bash
httptest -url "https://example.com" -requests 1000 -concurrency 100
```

### 2. Duration-Based POST Request with JSON Body from File

Run a test for 30 seconds, sending `POST` requests with a JSON body loaded from `body.json` and including a custom `Authorization` header:

```bash
# First, create your body.json file (e.g., echo '{"key":"value"}' > body.json)

httptest -url "https://api.example.com/v1/data" \
           -duration 30s \
           -concurrency 20 \
           -method "POST" \
           -body-file "body.json" \
           -header "Content-Type: application/json" \
           -header "Authorization: Bearer your_auth_token_here"
```

### 3. Save Test Summary to a File

Run a test and save the comprehensive final report to `report.json` for later analysis or integration with other tools:

```bash
httptest -url "https://example.com" -requests 500 -output report.json
```

## Building from Source (For Developers)

If you plan to contribute to `httptest` or want to build and run it directly from the source code, follow these steps:

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/saransridatha/httptest.git
    cd httptest
    ```
2.  **Build the executable:**
    ```bash
    go build script.go
    ```
3.  **Run it from the local directory:**
    ```bash
    ./script -url "https://example.com" -requests 100
    ```

## Contributing

We welcome contributions to `httptest`! If you have suggestions, bug reports, or would like to contribute code, please feel free to open an issue or submit a pull request on the GitHub repository.

## License

This project is licensed under the MIT License.

---
