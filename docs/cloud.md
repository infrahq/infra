# Infra Cloud

### Goals
* One place for teams to manage 2+ Infra Engines
* Single source of truth for data that cannot be implemented as code / configuration (i.e. users, groups, audit logs, etc)
* Central sink & index for audit log data
* Minimize risk for customers to use Infra Cloud
* Non-goal: proxy all customer data through our own environment
* Non-goal: single source of failure for customers
* Non-goal: Infra Cloud does not need access to customer environment (think Datadog agent)

### High-level architecture
![cloud architecture](https://user-images.githubusercontent.com/251292/114213582-22ba1600-9931-11eb-9ea7-b4edd516a5da.png)


### Pricing
* Usage-based pricing based on amount of data AND/OR number of identities managed (TBD)
* E.g. $x/identity or $y/million events audited

### User Experience
* See [Figma mockup](https://www.figma.com/file/WjpyKmfMHeUapLDWRVYb1G/Cloud-User-Flow?node-id=0%3A1)
* `infra login` logs in to Infra Cloud by default

### Open Questions
* Alternative to Infra Cloud is building your own version with the Infra Engine API – is this a good model?
* Pricing model?
* Security/risk model for customers?
* Defensibility (vs AWS)
* Est. costs?
