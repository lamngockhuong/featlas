# Ruby on Rails

- **Routes**: `config/routes.rb` (`resources :claims`, `namespace :admin`). `bin/rails routes` prints the
  table when you can run it. `resources` expands to index/show/new/create/edit/update/destroy.
- **Screens**: controller actions render `app/views/<controller>/<action>.html.erb` (or Hotwire/React
  components). Partials and Turbo frames show what updates in place.
- **Roles**: `before_action :authenticate_user!`, Pundit policies (`app/policies`), CanCanCan abilities.
- **Validation and rules**: model validations, callbacks and state machines (AASM) in `app/models`.
- **Data**: `db/schema.rb` lists tables and columns; `db/seeds.rb` demo accounts. Jobs in `app/jobs`.
