# Resource: honeycombio_board

Creates a board. For more information about boards, check out [Collaborate with Boards](https://docs.honeycomb.io/working-with-your-data/collaborating/boards/#docs-sidebar).

## Example Usage

```hcl
variable "dataset" {
  type = string
}

locals {
  percentiles = ["P50", "P75", "P90", "P95"]
}

data "honeycombio_query_specification" "query" {
  for_each = toset(local.percentiles)

  calculation {
    op     = local.percentiles[each.key]
    column = "duration_ms"
  }

  filter {
    column = "trace.parent_id"
    op     = "does-not-exist"
  }

  filter {
    column = "app.tenant"
    op     = "="
    value  = "ThatSpecialTenant"
  }
}

resource "honeycombio_query" "query" {
  for_each = toset(local.percentiles)

  dataset    = var.dataset
  query_json = data.honeycombio_query_specification.query[each.key].json
}

resource "honeycombio_board" "board" {
  name        = "Request percentiles"
  description = "${join(", ", local.percentiles)} of all requests for ThatSpecialTenant for the last 15 minutes."
  style       = "list"

  //Use dynamic config blocks to generate a query for each of the percentiles we're interested in
  dynamic "query" {
    for_each = local.percentiles

    content {
      caption     = query.value
      query_style = "combo"
      dataset     = var.dataset
      query_json  = data.honeycombio_query_specification.query[query.key].json
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the board.
* `description` - (Optional) Description of the board. Supports markdown.
* `style` - (Optional) How the board should be displayed in the UI, either `list` (the default) or `visual`.
* `query` - (Optional) Zero or more configurations blocks (described below) with the queries of the board.

Each board configuration may have zero or more `query` blocks, which accepts the following arguments:

* `query_id` - (Required) The ID of the Query to run.
* `query_annotation_id` - (Optional) The ID of the Query Annotation to associate with this query.
* `dataset` - (Required) The dataset this query is associated with.
* `caption` - (Optional) A description of the query that will be displayed on the board. Supports markdown.
* `query_style` - (Optional) How the query should be displayed within the board, either `graph` (the default), `table` or `combo`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - ID of the board.

## Import

Boards can be imported using their ID, e.g.

```
$ terraform import honeycombio_board.my_board AobW9oAZX71
```

You can find the ID in the URL bar when visiting the board from the UI.
