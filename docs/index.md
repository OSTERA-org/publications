---
layout: home
title: Home
hero_title: "OSTERA Publications"
hero_text: "Approved documents published by OSTERA. Documents are developed in OSTERA Working Groups and are freely available to everyone."
hero_buttons:
  - label: Working Groups
    url: https://wg.ostera.org/
  - label: About These Documents
    url: /about/
    style: secondary
---

## Documents by Working Group

{% for wg in site.data.working_groups -%}
{%- assign docs = site.data.documents | where: "wg", wg.id -%}
{%- if docs.size > 0 %}
### [{{ wg.name }}](https://wg.ostera.org/{{ wg.id }}/)

| Document | Type | Latest Version |
| --- | --- | --- |
{% for doc in docs -%}
| [{{ doc.title }}]({{ '/' | append: doc.id | append: '/' | relative_url }}){% if doc.number != "" %}<br>{{ doc.number }}{% endif %} | {{ doc.type }} | {% if doc.versions.size > 0 %}{% assign latest = doc.versions | first %}[Version {{ latest.version }}]({{ '/' | append: doc.id | append: '/v' | append: latest.version | append: '/' | relative_url }}) ({{ latest.date }}){% else %}Not yet published{% if doc.status != "" %} ({{ doc.status }}){% endif %}{% endif %} |
{% endfor %}
{%- endif %}
{% endfor %}

Document type codes are explained on [About These Documents]({{ '/about/' | relative_url }}).
