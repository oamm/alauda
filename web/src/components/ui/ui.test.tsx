import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import {
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  StatusBadge,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  FormField,
  Input,
  PasswordInput,
  MultiSelect,
  Section,
  Stat,
  StatGroup,
  ActivityList,
} from ".";

describe("Alauda UI primitives", () => {
  it("groups summary metrics and names sections without repeating context", async () => {
    const user = userEvent.setup();
    let opened = false;
    render(
      <>
        <StatGroup label="Service summary">
          <Stat label="Instances" value={2} />
          <Stat label="Endpoints" value={3} />
        </StatGroup>
        <Section
          title="Monitoring"
          action={
            <Button
              onClick={() => {
                opened = true;
              }}
            >
              Configure health
            </Button>
          }
        >
          No checks configured
        </Section>
      </>,
    );
    expect(
      screen.getByRole("group", { name: "Service summary" }),
    ).toHaveTextContent("Instances2Endpoints3");
    expect(
      screen.getByRole("region", { name: "Monitoring" }),
    ).toHaveTextContent("No checks configured");
    await user.click(screen.getByRole("button", { name: "Configure health" }));
    expect(opened).toBe(true);
  });

  it("keeps activity time, event and context together in source order", () => {
    render(
      <ActivityList
        items={[
          {
            id: "first",
            time: "Today",
            event: "Incident opened",
            context: "Instance unhealthy",
          },
          {
            id: "second",
            time: "Yesterday",
            event: "Instance added",
            context: "Production",
          },
        ]}
      />,
    );
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("TodayIncident openedInstance unhealthy");
    expect(rows[1]).toHaveTextContent("YesterdayInstance addedProduction");
  });
  it("restores focus after a controlled dialog closes with Escape", async () => {
    const user = userEvent.setup();
    function ControlledDialog() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <Button onClick={() => setOpen(true)}>Open form</Button>
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Edit resource</DialogTitle>
              </DialogHeader>
              <DialogBody>
                <Input aria-label="Name" />
              </DialogBody>
            </DialogContent>
          </Dialog>
        </>
      );
    }
    render(<ControlledDialog />);
    const trigger = screen.getByRole("button", { name: "Open form" });
    await user.click(trigger);
    expect(screen.getByRole("textbox", { name: "Name" })).toHaveFocus();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });
  it("connects field labels, validation errors and helper text", () => {
    const { rerender } = render(
      <FormField label="Hostname" hint="Use a DNS name">
        <Input />
      </FormField>,
    );
    const input = screen.getByRole("textbox", { name: "Hostname" });
    expect(input).toHaveAccessibleDescription("Use a DNS name");
    rerender(
      <FormField label="Hostname" error="Hostname is required">
        <Input />
      </FormField>,
    );
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(input).toHaveAccessibleDescription("Hostname is required");
    expect(screen.getByRole("alert")).toHaveTextContent("Hostname is required");
  });

  it("reveals passwords without submitting a form or changing the value", async () => {
    const user = userEvent.setup();
    render(
      <form>
        <FormField label="Password">
          <PasswordInput defaultValue="secret" />
        </FormField>
      </form>,
    );
    const input = screen.getByLabelText("Password");
    await user.click(screen.getByRole("button", { name: "Show password" }));
    expect(input).toHaveAttribute("type", "text");
    expect(input).toHaveValue("secret");
    expect(
      screen.getByRole("button", { name: "Hide password" }),
    ).toHaveAttribute("type", "button");
    await user.click(screen.getByRole("button", { name: "Hide password" }));
    expect(input).toHaveAttribute("type", "password");
  });

  it("supports multiple selections with unique accessible option IDs", async () => {
    const user = userEvent.setup();
    function Choices() {
      const [value, setValue] = useState<string[]>([]);
      return (
        <FormField label="Channels">
          <MultiSelect
            options={[
              { value: "email", label: "Email" },
              { value: "webhook", label: "Webhook" },
            ]}
            value={value}
            onValueChange={setValue}
          />
        </FormField>
      );
    }
    render(<Choices />);
    await user.click(screen.getByRole("button", { name: "Channels" }));
    const email = screen.getByRole("checkbox", { name: "Email" });
    const webhook = screen.getByRole("checkbox", { name: "Webhook" });
    expect(email.id).not.toBe(webhook.id);
    await user.click(email);
    await user.click(webhook);
    expect(email).toBeChecked();
    expect(webhook).toBeChecked();
    await user.keyboard("{Escape}");
    expect(screen.getByRole("button", { name: "Channels" })).toHaveTextContent(
      "Email, Webhook",
    );
  });
  it("exposes a canonical button and status language", () => {
    render(
      <>
        <Button variant="primary">Create service</Button>
        <StatusBadge status="HEALTH_STATE_HEALTHY" />
      </>,
    );
    expect(screen.getByRole("button", { name: "Create service" })).toHaveClass(
      "bg-[var(--accent)]",
    );
    expect(screen.getByText("Healthy")).toBeInTheDocument();
  });

  it("supports keyboard-accessible tabs", async () => {
    const user = userEvent.setup();
    render(
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="events">Events</TabsTrigger>
        </TabsList>
        <TabsContent value="overview">Summary</TabsContent>
        <TabsContent value="events">History</TabsContent>
      </Tabs>,
    );
    await user.click(screen.getByRole("tab", { name: "Events" }));
    expect(screen.getByText("History")).toBeVisible();
  });

  it("provides a structured dialog surface", () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create service</DialogTitle>
          </DialogHeader>
          <DialogBody>Form body</DialogBody>
          <DialogFooter>
            <Button>Cancel</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>,
    );
    expect(screen.getByRole("dialog")).toHaveTextContent("Create service");
    expect(screen.getByRole("dialog")).toHaveTextContent("Form body");
  });
});
