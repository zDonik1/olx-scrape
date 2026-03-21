# OLX Scraper

A CLI tool for scraping listings from [olx.uz](https://www.olx.uz), extracting structured data, and exporting results to Excel.

> [!NOTE]
> Some of the code is written only for PC listings, so it doesn't work out of the box for other types of listings. Further work can be done to generalize to other listings.

## ✨ Features

- 🔍 **Category-based scraping** - target any OLX category by passing its URL path (e.g. `elektronika/kompyutery/nastolnye`)
- 🧩 **Structured data extraction** - automatically parses listing ID, date, price, condition, title, and description
- 💱 **Currency normalization** - converts UZS (som) prices to USD using a fixed exchange rate
- 🤖 **AI-powered component parsing** - optionally uses a local Ollama LLM to extract individual PC parts (CPU, GPU, RAM, storage, motherboard, cooler, case, PSU, OS) from listing text
- 📊 **Excel export** - writes all results to a formatted `output.xlsx` file with proper column headers and currency styling
- 💾 **Caching** - saves fetched HTML pages and AI-processed structured data locally to avoid redundant network requests and LLM calls
- 🔄 **Cache invalidation** - selectively refresh the full cache or just the listing-pages cache via flags
- 🚦 **Ad limit** - cap the number of ads processed with `--max-ads` to control run size
- ⚡ **Concurrent pipeline** - pages, ads, and AI processing run in separate goroutine worker pools connected by channels, where the number of workers can be configured

## 🚀 Usage

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--category` | `-c` | `""` | OLX category path to scrape |
| `--pages` | `-p` | `1` | Number of listing pages to scan |
| `--max-ads` | | `0` | Max ads to process (0 = unlimited) |
| `--ai-processing` | `-a` | `false` | Enable AI component extraction via Ollama |
| `--refresh-cache` | `-R` | `false` | Clear and rebuild all cached HTML |
| `--refresh-pages-cache` | `-P` | `false` | Clear and rebuild only pages cache |
| `--verbose` | `-v` | `false` | Enable debug logging |

### Example

```bash
# Scrape 3 pages of desktop PCs with AI processing enabled
go run . -c elektronika/kompyutery/nastolnye -p 3 -a
```

## 📦 Output

Results are saved to `output.xlsx` with the following columns:

`id` · `date` · `price` · `condition` · `name` · `description` · `url` · `cpu` · `gpu` · `ram` · `storage` · `motherboard` · `cpu_cooler` · `case` · `psu` · `os`

## 🛠 Requirements

- [Go 1.25+](https://go.dev/)
- [Ollama](https://ollama.com/) with the `gemma3n` model (only required for `-a` flag)
