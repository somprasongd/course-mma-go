create table public.customers (
  id serial PRIMARY KEY,
  name text NOT NULL,
  credit integer NOT NULL,
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);