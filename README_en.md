[日本語版](README.md)

# microblogen
Jamstack blog generator with microCMS

## Overview
`microblogen` is a Go tool that retrieves article data from [microCMS](https://microcms.io/)
and generates a static blog by inserting it into templates.

## Installation
You can build it by running the following command in an environment where Go 1.20 or later is installed.

```bash
go build
```

## Usage
Set the required values as environment variables, then execute as follows:

```bash
./microblogen
```

### Available Environment Variables
| Variable Name | Description | Default |
| ------ | ---- | ---------- |
| `MICROCMS_API_KEY` | API key for microCMS (required) | - |
| `SERVICE_DOMAIN` | Service domain for microCMS (required) | - |
| `EXPORT_PATH` | Output directory (optional) | `./output` |
| `TEMPLATE_PATH` | Template directory (optional) | `./template` |
| `PAGE_SHOW_LIMIT` | Number of articles to display per page (optional) | `10` |
| `TIMEZONE` | Timezone for dates (optional) | `UTC` |
| `CATEGORY_TAG_NAME` | Label for category display (optional) | `Category` |
| `TIME_ARCHIVE_NAME` | Label for archive display (optional) | `Archive` |

## License
The code in this repository is provided under the [MIT License](LICENSE).
