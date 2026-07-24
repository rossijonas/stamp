---
---

# Architecture Decisions

Architecture Decision Records (ADRs) document key technical decisions made during the development of Stamp.

{% assign adrs = site.pages | where_exp: "p", "p.dir contains 'decisions/'" | sort: "name" | reverse %}
{% for adr in adrs %}
{% if adr.name != "index.md" %}
- [{{ adr.name | replace: "-", " " | replace: ".md", "" | capitalize }}]({{ adr.url | relative_url }})
{% endif %}
{% endfor %}
