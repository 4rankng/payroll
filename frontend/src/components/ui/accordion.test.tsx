import { fireEvent, render, screen } from "@testing-library/react"

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "./accordion"

describe("AccordionTrigger", () => {
  it("exposes button semantics and opens from the keyboard", () => {
    render(
      <Accordion type="single" collapsible>
        <AccordionItem value="details">
          <AccordionTrigger>Chi tiết</AccordionTrigger>
          <AccordionContent>Nội dung chi tiết</AccordionContent>
        </AccordionItem>
      </Accordion>,
    )

    const trigger = screen.getByRole("button", { name: "Chi tiết" })
    expect(trigger).toHaveAttribute("aria-expanded", "false")

    fireEvent.keyDown(trigger, { key: "Enter" })

    expect(trigger).toHaveAttribute("aria-expanded", "true")
  })
})

describe("Accordion UU contract (W7)", () => {
  it("divider sits on utility-gray-200 and trigger typography uses UU fg tokens", () => {
    render(
      <Accordion type="single" collapsible>
        <AccordionItem value="details" data-testid="item">
          <AccordionTrigger>Chi tiết</AccordionTrigger>
        </AccordionItem>
      </Accordion>,
    )

    const item = screen.getByTestId("item")
    const trigger = screen.getByRole("button", { name: "Chi tiết" })

    expect(item).toHaveClass("border-b", "border-utility-gray-200")
    expect(trigger).toHaveClass("font-semibold", "text-fg-primary")
  })

  it("chevron carries the fg-tertiary token", () => {
    render(
      <Accordion type="single" collapsible>
        <AccordionItem value="details">
          <AccordionTrigger>Chi tiết</AccordionTrigger>
        </AccordionItem>
      </Accordion>,
    )

    const trigger = screen.getByRole("button", { name: "Chi tiết" })
    const chevron = trigger.querySelector("svg")
    expect(chevron).toHaveClass("text-fg-tertiary")
  })

  it("lets caller classes override trigger classes through the merge", () => {
    render(
      <Accordion type="single" collapsible>
        <AccordionItem value="details">
          <AccordionTrigger className="py-2">Chi tiết</AccordionTrigger>
        </AccordionItem>
      </Accordion>,
    )

    const trigger = screen.getByRole("button", { name: "Chi tiết" })
    expect(trigger).toHaveClass("py-2")
    expect(trigger).not.toHaveClass("py-4")
  })
})
