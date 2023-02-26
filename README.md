## What is it?
- Using `chromedp` to login Vanguard ISA and extract investments values
- It demos
** form submission
** taking screenshots
** extracting text values from table
** posting values to google sheet

## How to run?
- create `urls.json` with template given and fill in the values
```bash
cp urls.json.template urls.json
cp .env.template .env
```
- run the program
```bash
go run .
```
