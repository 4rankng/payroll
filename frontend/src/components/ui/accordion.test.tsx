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
