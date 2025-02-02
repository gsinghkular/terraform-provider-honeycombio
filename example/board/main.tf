terraform {
  required_providers {
    honeycombio = {
      source = "honeycombio/honeycombio"
    }
  }
}

variable "dataset" {
  type = string
}

locals {
  percentiles = ["P50", "P75", "P90", "P95"]
}

data "honeycombio_query_specification" "query" {
  for_each = toset(local.percentiles)

  calculation {
    op     = local.percentiles[count.index]
    column = "duration_ms"
  }

  filter {
    column = "trace.parent_id"
    op     = "does-not-exist"
  }
  filter {
    column       = "app.tenant"
    op           = "in"
    value_string = "foo,bar" # op 'in' expects a list of values
  }
}

resource "honeycombio_query" "query" {
  for_each = to_set(local.percentiles)

  dataset    = var.dataset
  query_json = data.honeycombio_query_specification.query[each.key].json
}

resource "honeycombio_board" "board" {
  name        = "Request percentiles"
  description = "${join(", ", local.percentiles)} of all requests for ThatSpecialTenant for the last 15 minutes."
  style       = "list"

  # Use dynamic config blocks to generate a query for each of the percentiles we're interested in
  dynamic "query" {
    for_each = local.percentiles

    content {
      caption = query.value
      dataset = var.dataset
      query_id = honeycombio_query.query[query.key].id
    }
  }
}
