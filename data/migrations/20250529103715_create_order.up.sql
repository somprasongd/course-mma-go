create table public.orders (
  id serial PRIMARY KEY,
  customer_id integer NOT NULL,
  order_total integer NOT NULL,
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  canceld_at timestamp,
  CONSTRAINT fk_customer
      FOREIGN KEY(customer_id)
        REFERENCES customers(id)
);