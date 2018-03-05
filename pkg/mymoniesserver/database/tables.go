package database

type table struct {
	name   string
	create string
	drop   string
}

var tables = []table{
	{
		name: "imports",
		create: `
			CREATE TABLE IF NOT EXISTS imports (
				id serial UNIQUE,
				filename text,
				account text NOT NULL
			);
		`,
		drop: "DROP TABLE IF EXISTS imports",
	},

	{
		name: "tags",
		create: `
			CREATE TABLE IF NOT EXISTS tags (
				id serial UNIQUE,
				name text UNIQUE
			);
		`,
		drop: "DROP TABLE IF EXISTS tags",
	},

	{
		name: "records",
		create: `
			CREATE TABLE IF NOT EXISTS records (
				id serial UNIQUE,
				import_id int REFERENCES imports(id) ON DELETE CASCADE,
				transaction_date date ,
				value_date date,
				payment_date date,
				amount double precision,
				payee_payer text,
				account text,
				bic text,
				transaction text,
				reference text,
				payer_reference text,
				message text,
				card_number text,
				tag_id int REFERENCES tags(id)
			);
			`,
		drop: "DROP TABLE IF EXISTS records",
	},

	{
		name: "patterns",
		create: `
			CREATE TABLE IF NOT EXISTS patterns (
				id serial		UNIQUE,
				tag_id			int REFERENCES tags(id),
				account			text NOT NULL,
				query			text NOT NULL
			);
		`,
		drop: "DROP TABLE IF EXISTS patterns",
	},
}
