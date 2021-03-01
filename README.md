# Reddit Stock Webscrapper

## What it does

1. Scrap /r/wallstreetbets for yesterday's Daily Discussion
2. Gather all comment ids
3. Use [pushshift API](https://github.com/pushshift/api) to get comment text
4. Count stock ticket mentions in all comments
5. Write results to a csv
6. Move csv to S3 bucket

## TODO

- Gather current stock tickets on a monthly? interval from [dumbstockapi](https://dumbstockapi.com/)
- Write to database
