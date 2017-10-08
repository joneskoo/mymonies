# mymonies

mymonies is an utility to tag bank transaction records and credit card
statements by spend category.

Development status: proof-of-concept.

## Features

* mymonies-import (command-line)
    * Import transaction records to PostgreSQL database
    * Supported data formats: Nordea Bank account TSV
    * Set default tag of records by pre-defined rules (JSON pattern configuration)
* mymonies (web interface)
    * List accounts
    * List transactions by account
    * Update missing or incorrect tag, dropdown selection
