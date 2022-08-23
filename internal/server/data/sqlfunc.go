package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data/migrator"
)

func sqlFunctionsMigration() *migrator.Migration {
	return &migrator.Migration{
		ID: "2022-08-22T14:58:00Z",
		Migrate: func(db *gorm.DB) error {
			sql := `
			CREATE FUNCTION uidIntToStr(id BIGINT) RETURNS text
			LANGUAGE PLPGSQL
			AS $$
			DECLARE
			encodebase58map CONSTANT TEXT := '123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ';
			base_count 			BIGINT DEFAULT 0;
				encoded    			TEXT DEFAULT '';
				divisor    			BIGINT;
				mod        			BIGINT DEFAULT 0;
			
			BEGIN
				IF id <= 0 THEN
					RETURN '';
				END IF;

				IF id < 58 THEN
					RETURN SUBSTRING(encodeBase58Map FROM id FOR 1);
				END IF;

				WHILE id > 0 LOOP
					divisor := id / 58;
					mod := (id - (58 * divisor));
					encoded = CONCAT(SUBSTRING(encodeBase58Map FROM CAST(mod+1 as int) FOR 1), encoded);
					id = id / 58;
				END LOOP;
			
				RETURN encoded;
			
			END; $$;

			CREATE FUNCTION uidStrToInt(encoded TEXT) RETURNS BIGINT
			LANGUAGE PLPGSQL
			AS $$
			DECLARE
				encodebase58map CONSTANT TEXT := '123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ';
				base_count 			BIGINT := 0;
				result    			BIGINT := 0;
				i    						INT := 1;
				pos        			BIGINT := 0;
				MAX_SIGNED_INT CONSTANT BIGINT := 9223372036854775807;
			
			BEGIN
				WHILE (SUBSTRING(encoded FROM 1 FOR 1) = '1') LOOP
					encoded = SUBSTRING(encoded FROM 2 FOR LENGTH(encoded)-1);
				END LOOP;

				IF LENGTH(encoded) > 11 THEN
					RAISE EXCEPTION 'invalid base58: too long';
				END IF;

				WHILE (i <= LENGTH(encoded)) LOOP
				  IF (result > MAX_SIGNED_INT/58) THEN 
						RAISE EXCEPTION 'invalid base58: value too large: %', encoded;
					END IF;
					result = result * 58;
					pos := POSITION(SUBSTRING(encoded FROM i FOR 1) in encodeBase58Map);
					IF (pos <= 0) THEN
						RAISE EXCEPTION 'invalid base58: byte % is out of range', i-1;
					END IF;
					IF (MAX_SIGNED_INT - (pos - 1) < result) THEN
						RAISE EXCEPTION 'invalid base58: value too large: %', encoded;
					END IF;
					result = result + (pos-1);
					i = i + 1;
				END LOOP;

				RETURN result;
			
			END; $$;
			`
			return db.Exec(sql).Error

		},
	}
}
