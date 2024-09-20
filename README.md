## Overview

This is a simple tray menu application designed to help manage and control services. It provides a lightweight interface to create and run services easily from the system tray.

## Features

- Add Service: Quickly create and configure your service.
- Run Service: Start your service using a bash -c command.

## How to Use

1. Add a New Service
   Click on Add Service in the tray menu to create a new service. You will need to provide the necessary command or script to run the service.

2. Run the Service
   After creating a service, select it from the tray menu and choose Run. The service will be executed using the command bash -c.

## Requirements

- Bash (for executing services)
- A compatible operating system that supports tray menus (Linux, macOS, etc.)

## Installation

1. Install from the repository:

```
 go install github.com/xxlv/go-servicemanager@latest
```


## License

This project is licensed under the MIT License.
