# frozen_string_literal: true
#
# Pure-Ruby usage of the PublicSuffix module, as provided by go-embedded-ruby (rbgo).
# Run it with:  rbgo examples/public_suffix_usage.rb

require "public_suffix"

# Registrable domain (sld.tld), or nil when the name has none.
puts PublicSuffix.domain("www.example.co.uk") # => example.co.uk

# Full decomposition into a PublicSuffix::Domain.
d = PublicSuffix.parse("www.example.co.uk")
p [d.tld, d.sld, d.trd] # => ["co.uk", "example", "www"]
puts d.domain           # => example.co.uk
puts d.subdomain        # => www.example.co.uk

# Validation against the Public Suffix List.
p PublicSuffix.valid?("example.com") # => true

# Strict checking: default_rule: nil disables the "*" fallback for unlisted TLDs.
p PublicSuffix.valid?("example.tldnotlisted", default_rule: nil) # => false

# ignore_private: true skips the PRIVATE DOMAINS section.
pd = PublicSuffix.parse("foo.blogspot.com", ignore_private: true)
puts pd.domain # => blogspot.com
