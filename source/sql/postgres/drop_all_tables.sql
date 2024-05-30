-- https://stackoverflow.com/a/36023359/9731532
DO
$$
    DECLARE r RECORD;
    BEGIN
        -- if the schema you operate on is not "current", you will want to
        -- replace current_schema() in query with 'schematodeletetablesfrom'
        -- *and* update the generate 'DROP...' accordingly.
        FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
            EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
        END LOOP;

        -- Delete all types too
        FOR r IN (SELECT typname FROM pg_type WHERE typnamespace = 'public'::regnamespace) LOOP
            EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
        END LOOP;
    END
$$;
