INSERT INTO tags (name) VALUES ('example');
INSERT INTO tags (name) VALUES ('example2');
INSERT INTO imports (filename, account) VALUES ('asdf', 'foo');
INSERT INTO records
        (
                import_id,
                transaction_date,
                value_date,
                payment_date,
                amount,
                payee_payer,
                account,
                bic,
                transaction,
                reference,
                payer_reference,
                message,
                card_number,
                tag_id
        ) VALUES (
                1,
                '2018-03-01'::date,
                '2018-03-02'::date,
                '2018-03-03'::date,
                10,
                'payee or payer',
                'account number',
                'BIC value',
                'transaction id',
                'reference',
                'payer reference',
                'message',
                'card number',
                1
        );
