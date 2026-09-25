create table if not exists wallets (
	id uuid primary key,
	balance bigint not null default 0 constraint balance_non_negative check (balance >= 0),
	created_at timestamptz not null default current_timestamp
);
