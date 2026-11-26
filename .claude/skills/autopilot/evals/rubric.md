# Grounding rubric (LLM judge)

Use this to judge what `social_copy.py` can't: whether a post says things its
source item doesn't support (#321), and whether two posts on different days
cover the same story.

Judge **as a sceptical editor, not as the author**. If you wrote or changed
the prompt that produced these posts in this run, say so at the top of the
report. Your score still counts, but Ainsley should know.

## Inputs

For each post, use `social_post_metrics` for the text, then `get_issue` for
the issue's items, or `get_item` / `search_items` for the item. The **only**
evidence is the item's `title`, `snippet` and `url`. Do not fetch the linked
page: the drafting model never sees it either, so a claim that is only true on
the page is still unsupported.

## Per-post score

Read the post sentence by sentence. Label each factual claim:

| Label | Meaning |
|---|---|
| `supported` | Stated or directly implied by the title or snippet. |
| `general` | General Go knowledge that is true regardless of the item (e.g. "context.Context carries deadlines"). |
| `unsupported` | Specific to this item but not in the title or snippet (e.g. "the talk walks through concrete examples" when the snippet says nothing about examples). |
| `wrong` | Contradicts the title or snippet. |

Post verdict:
- **pass**: zero `unsupported` and zero `wrong`.
- **soft fail**: exactly one `unsupported` that is a framing claim about the
  content itself ("walks through", "covers migration").
- **fail**: any `wrong`, or two or more `unsupported`.

## Repeated stories

For featured posts, list pairs within 14 days that cover the **same story or
announcement**, even with different links (e.g. go.dev/blog/jsonv2 on 5 Nov
and a JSON v2 video on 16 Nov). A different angle on the same release still
counts as a repeat. A follow-up with new facts (e.g. the RC, then the final
release) does not.

## Report

Sample: the most recent 15 featured posts (LinkedIn version, the longest
text) plus every recap in the window.

```
grounding_pass_rate: 0.xx   (pass / judged)
grounding_fail_rate: 0.xx   (fail / judged)
repeated_stories_14d: N
worst examples: post id, claim, why it's unsupported (max 3)
```

When comparing a prompt change, judge **before** and **after** with the same
sample size, and report both. Never report only the after.
