# Ruby examples

Pure-Ruby examples for the `public_suffix` library as provided by
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) (rbgo). Run them
with the `rbgo` interpreter:

```sh
rbgo examples/public_suffix_usage.rb
```

| File | Shows |
| --- | --- |
| [`public_suffix_usage.rb`](public_suffix_usage.rb) | Registrable domain, full decomposition, validation, and the `default_rule:` / `ignore_private:` options. |

Each example is executed as-is under rbgo (`require "public_suffix"`).
