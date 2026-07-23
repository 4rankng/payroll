"use client";

import { useMemo, useState } from "react";
import {
    ArrowLeft,
    BarChartSquare02,
    CheckDone01,
    ChevronDown,
    Copy01,
    CurrencyDollarCircle,
    Edit01,
    Eye,
    FilterLines,
    HomeLine,
    LayoutAlt01,
    MessageChatCircle,
    PieChart03,
    Rows01,
    SearchLg,
    Settings01,
    Share04,
    Stars02,
    Trash01,
    UserCircle,
    Users01,
} from "@untitledui/icons";
import type { SortDescriptor } from "react-aria-components";
import { Area, AreaChart, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer, XAxis } from "recharts";
import { FeaturedCardQRCode } from "@/components/application/app-navigation/base-components/featured-cards";
import { SidebarNavigationSimple } from "@/components/application/app-navigation/sidebar-navigation/sidebar-simple";
import { Breadcrumbs } from "@/components/application/breadcrumbs/breadcrumbs";
import { ChartTooltipContent } from "@/components/application/charts/charts-base";
import { MetricChangeIndicator, MetricsIcon04 } from "@/components/application/metrics/metrics";
import { PaginationPageMinimalCenter } from "@/components/application/pagination/pagination";
import { Table } from "@/components/application/table/table";
import { TabList, Tabs } from "@/components/application/tabs/tabs";
import { Avatar } from "@/components/base/avatar/avatar";
import { BadgeWithDot } from "@/components/base/badges/badges";
import { Button } from "@/components/base/buttons/button";
import { ButtonUtility } from "@/components/base/buttons/button-utility";
import { DropdownIconSimple } from "@/components/base/dropdown/dropdown-icon-simple";
import { InputBase } from "@/components/base/input/input";

// Helper functions for formatting
const formatCurrency = (amount: number): string => amount.toLocaleString("en-US", { style: "currency", currency: "USD" });

const formatDate = (timestamp: number): string =>
    new Date(timestamp).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    });

const lineData = [
    {
        date: new Date(2026, 0, 1),
        A: 600,
        B: 400,
    },
    {
        date: new Date(2026, 1, 1),
        A: 620,
        B: 405,
    },
    {
        date: new Date(2026, 2, 1),
        A: 630,
        B: 400,
    },
    {
        date: new Date(2026, 3, 1),
        A: 650,
        B: 410,
    },
    {
        date: new Date(2026, 4, 1),
        A: 600,
        B: 320,
    },
    {
        date: new Date(2026, 5, 1),
        A: 650,
        B: 430,
    },
    {
        date: new Date(2026, 6, 1),
        A: 620,
        B: 400,
    },
    {
        date: new Date(2026, 7, 1),
        A: 750,
        B: 540,
    },
    {
        date: new Date(2026, 8, 1),
        A: 780,
        B: 490,
    },
    {
        date: new Date(2026, 9, 1),
        A: 750,
        B: 450,
    },
    {
        date: new Date(2026, 10, 1),
        A: 780,
        B: 480,
    },
    {
        date: new Date(2026, 11, 1),
        A: 820,
        B: 500,
    },
];

const customers = [
    {
        name: "Lily-Rose Chedjou",
        email: "lilyrose@gmail.com",
        username: "@lilyrose",
        avatarUrl: "https://www.untitledui.com/images/avatars/lily-rose-chedjou?fm=webp&q=80",
        date: new Date(2026, 0, 16).getTime(),
        status: "paid",
        amount: 100.14,
    },
    {
        name: "Caitlyn King",
        email: "hi@caitlynking.com",
        username: "@caitlynk",
        avatarUrl: "https://www.untitledui.com/images/avatars/caitlyn-king?fm=webp&q=80",
        date: new Date(2026, 0, 16).getTime(),
        status: "paid",
        amount: 96.32,
    },
    {
        name: "Fleur Cook",
        email: "fleurcook@icloud.com",
        username: "@fleur_cook",
        avatarUrl: "https://www.untitledui.com/images/avatars/fleur-cook?fm=webp&q=80",
        date: new Date(2026, 0, 15).getTime(),
        status: "paid",
        amount: 104.24,
    },
    {
        name: "Marco Kelly",
        email: "marco@marcokelly.co",
        username: "@marcokelly",
        avatarUrl: "https://www.untitledui.com/images/avatars/marco-kelly?fm=webp&q=80",
        date: new Date(2026, 0, 14).getTime(),
        status: "paid",
        amount: 88.48,
    },
    {
        name: "Lulu Meyers",
        email: "lulu@lulumeyers.com",
        username: "@lulu_meyers",
        avatarUrl: "https://www.untitledui.com/images/avatars/lulu-meyers?fm=webp&q=80",
        date: new Date(2026, 0, 14).getTime(),
        status: "paid",
        amount: 96.32,
    },
    {
        name: "Mikey Lawrence",
        email: "m.lawrence@gmail.com",
        username: "@mikeylawrence",
        avatarUrl: "https://www.untitledui.com/images/avatars/mikey-lawrence?fm=webp&q=80",
        date: new Date(2026, 0, 14).getTime(),
        status: "paid",
        amount: 107.1,
    },
    {
        name: "Freya Browning",
        email: "hey@freyabrowning.com",
        username: "@freya_b",
        avatarUrl: "https://www.untitledui.com/images/avatars/freya-browning?fm=webp&q=80",
        date: new Date(2026, 0, 14).getTime(),
        status: "paid",
        amount: 82.04,
    },
];

const getInitials = (name: string) => {
    const [firstName, lastName] = name.split(" ");

    return `${firstName.charAt(0)}${lastName.charAt(0)}`;
};

const accounts = [
    {
        id: "caitlyn",
        name: "Caitlyn King",
        email: "caitlyn@untitledui.com",
        avatar: "https://www.untitledui.com/images/avatars/caitlyn-king?fm=webp&q=80",
        status: "online",
    },
    {
        id: "sienna",
        name: "Sienna Hewitt",
        email: "sienna@untitledui.com",
        avatar: "https://www.untitledui.com/images/avatars/transparent/sienna-hewitt?bg=%23E0E0E0",
        status: "online",
    },
];

export const Dashboard04 = () => {
    const [sortDescriptor, setSortDescriptor] = useState<SortDescriptor>();
    const [selectedAccountKey, setSelectedAccountKey] = useState<string>("caitlyn");

    const sortedItems = useMemo(() => {
        if (!sortDescriptor) return customers;

        return customers.toSorted((a, b) => {
            const first = a[sortDescriptor.column as keyof typeof a];
            const second = b[sortDescriptor.column as keyof typeof b];

            // Handle numbers
            if (typeof first === "number" && typeof second === "number") {
                return sortDescriptor.direction === "ascending" ? first - second : second - first;
            }

            // Handle strings
            if (typeof first === "string" && typeof second === "string") {
                const result = first.localeCompare(second);
                return sortDescriptor.direction === "ascending" ? result : -result;
            }

            return 0;
        });
    }, [sortDescriptor]);

    return (
        <div className="flex flex-col bg-primary lg:flex-row">
            <SidebarNavigationSimple
                items={[
                    {
                        label: "Home",
                        href: "/",
                        icon: HomeLine,
                        items: [
                            { label: "Overview", href: "/overview" },
                            { label: "Products", href: "/products" },
                            { label: "Orders", href: "/orders" },
                            { label: "Customers", href: "/customers" },
                        ],
                    },
                    {
                        label: "Dashboard",
                        href: "/dashboard",
                        icon: BarChartSquare02,
                        items: [
                            { label: "Overview", href: "/dashboard/overview" },
                            { label: "Notifications", href: "/dashboard/notifications", badge: 10 },
                            { label: "Analytics", href: "/dashboard/analytics" },
                            { label: "Saved reports", href: "/dashboard/saved-reports" },
                        ],
                    },
                    {
                        label: "Projects",
                        href: "/projects",
                        icon: Rows01,
                        items: [
                            { label: "View all", href: "/projects/all" },
                            { label: "Personal", href: "/projects/personal" },
                            { label: "Team", href: "/projects/team" },
                            { label: "Shared with me", href: "/projects/shared-with-me" },
                            { label: "Archive", href: "/projects/archive" },
                        ],
                    },
                    {
                        label: "Tasks",
                        href: "/tasks",
                        icon: CheckDone01,
                        badge: 8,
                        items: [
                            { label: "My tasks", href: "/tasks/my-tasks" },
                            { label: "Assigned to me", href: "/tasks/assigned" },
                            { label: "Completed", href: "/tasks/completed" },
                            { label: "Upcoming", href: "/tasks/upcoming" },
                        ],
                    },
                    {
                        label: "Reporting",
                        href: "/reporting",
                        icon: PieChart03,
                        items: [
                            { label: "Dashboard", href: "/reporting/dashboard" },
                            { label: "Revenue", href: "/reporting/revenue" },
                            { label: "Performance", href: "/reporting/performance" },
                            { label: "Export data", href: "/reporting/export" },
                        ],
                    },
                    {
                        label: "Users",
                        href: "/users",
                        icon: Users01,
                        items: [
                            { label: "All users", href: "/users/all" },
                            { label: "Admins", href: "/users/admins" },
                            { label: "Team members", href: "/users/team" },
                            { label: "Permissions", href: "/users/permissions" },
                        ],
                    },
                ]}
                footerItems={[
                    {
                        label: "Settings",
                        href: "/settings",
                        icon: Settings01,
                    },
                    {
                        label: "Support",
                        href: "/support",
                        icon: MessageChatCircle,
                        badge: (
                            <BadgeWithDot size="sm" color="success" type="modern">
                                Online
                            </BadgeWithDot>
                        ),
                    },
                    {
                        label: "Open in browser",
                        href: "https://www.google.com",
                        icon: LayoutAlt01,
                    },
                ]}
                featureCard={
                    <FeaturedCardQRCode
                        title="Verify this device"
                        description="Open the app and scan the QR code below to verify this device."
                        confirmLabel="Verify another way"
                        onConfirm={() => {}}
                        onDismiss={() => {}}
                    />
                }
            />

            <main className="min-w-0 flex-1 bg-primary pt-8 pb-12">
                <div className="flex flex-col gap-8">
                    <div className="flex flex-col gap-4 px-4 lg:px-8">
                        {/* Page header simple with avatar square */}
                        <div className="flex">
                            <Breadcrumbs className="max-lg:hidden">
                                <Breadcrumbs.Item href="#" avatarSrc="/logos/images/Warpspeed.jpg">
                                    Warpspeed
                                </Breadcrumbs.Item>

                                <Breadcrumbs.AccountItem items={accounts} selectedKey={selectedAccountKey} onSelectionChange={setSelectedAccountKey} />

                                <Breadcrumbs.Item>
                                    <span className="text-primary">Dashboard</span>
                                </Breadcrumbs.Item>
                            </Breadcrumbs>

                            <Button color="link-gray" iconLeading={ArrowLeft} size="xs" className="inline-flex lg:hidden">
                                Back
                            </Button>
                        </div>

                        <div className="flex flex-col items-start gap-4 lg:flex-row">
                            <p className="flex-1 text-xl font-semibold text-primary">My dashboard</p>
                            <div className="flex gap-3">
                                <Button size="sm" color="secondary" className="hidden lg:inline-flex" iconLeading={Stars02}>
                                    What's new?
                                </Button>
                                <Button size="sm" color="secondary" iconLeading={Copy01}>
                                    Copy link
                                </Button>
                                <Button size="sm" color="secondary" iconTrailing={Share04}>
                                    Visit store
                                </Button>
                            </div>
                        </div>
                    </div>

                    <div className="-my-2 flex flex-col gap-4 overflow-x-auto px-4 py-2 md:flex-row md:flex-wrap md:items-start md:gap-5 lg:px-8">
                        <MetricsIcon04
                            icon={CurrencyDollarCircle}
                            title="$8,746.22"
                            subtitle="All revenue"
                            change="2.4%"
                            changeTrend="positive"
                            actions={false}
                            className="flex-1 ring-2 ring-brand md:min-w-[320px]"
                        />
                        <MetricsIcon04
                            icon={Eye}
                            title="12,440"
                            subtitle="Page views"
                            change="6.2%"
                            changeTrend="positive"
                            actions={false}
                            className="flex-1 md:min-w-[320px]"
                        />
                        <MetricsIcon04
                            icon={UserCircle}
                            title="96"
                            subtitle="Active now"
                            change="0.8%"
                            changeTrend="positive"
                            actions={false}
                            className="flex-1 md:min-w-[320px]"
                        />
                    </div>

                    <div className="flex flex-col gap-5 px-4 lg:px-8">
                        <div className="flex flex-col items-start justify-between gap-4 lg:flex-row">
                            <div className="flex flex-col gap-2">
                                <div className="flex items-center gap-1">
                                    <span className="text-sm font-semibold text-tertiary">Net revenue</span>
                                    <ChevronDown size={16} className="text-fg-quaternary" />
                                </div>
                                <div className="flex items-center gap-3">
                                    <span className="text-display-sm font-semibold text-primary">$7,804.16</span>

                                    <MetricChangeIndicator type="modern" trend="positive" value="2.4%" />
                                </div>
                            </div>
                            <div className="flex gap-3">
                                <Tabs>
                                    <TabList
                                        type="button-minimal"
                                        items={[
                                            {
                                                id: "12months",
                                                label: (
                                                    <>
                                                        <span className="max-md:hidden">12 months</span>
                                                        <span className="md:hidden">12m</span>
                                                    </>
                                                ),
                                            },
                                            {
                                                id: "30days",
                                                label: (
                                                    <>
                                                        <span className="max-md:hidden">30 days</span>
                                                        <span className="md:hidden">30d</span>
                                                    </>
                                                ),
                                            },
                                            {
                                                id: "7days",
                                                label: (
                                                    <>
                                                        <span className="max-md:hidden">7 days</span>
                                                        <span className="md:hidden">7d</span>
                                                    </>
                                                ),
                                            },
                                            {
                                                id: "24hours",
                                                label: (
                                                    <>
                                                        <span className="max-md:hidden">24 hours</span>
                                                        <span className="md:hidden">24h</span>
                                                    </>
                                                ),
                                            },
                                        ]}
                                    />
                                </Tabs>

                                <Button size="sm" color="secondary" iconLeading={FilterLines}>
                                    Filters
                                </Button>
                            </div>
                        </div>

                        <div className="flex h-60 flex-col gap-2">
                            <ResponsiveContainer initialDimension={{ width: 1, height: 1 }} className="h-full">
                                <AreaChart data={lineData} className="text-tertiary [&_.recharts-text]:text-xs">
                                    <CartesianGrid vertical={false} stroke="currentColor" className="text-utility-neutral-100" />

                                    <XAxis
                                        fill="currentColor"
                                        axisLine={false}
                                        tickLine={false}
                                        interval="preserveStartEnd"
                                        dataKey="date"
                                        tickMargin={8}
                                        padding={{ left: 10, right: 10 }}
                                        tickFormatter={(value) => value.toLocaleDateString(undefined, { month: "short" })}
                                    />

                                    <RechartsTooltip
                                        content={<ChartTooltipContent />}
                                        formatter={(value) => value?.toLocaleString("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0 })}
                                        labelFormatter={(value) => value.toLocaleDateString(undefined, { month: "short", year: "numeric" })}
                                        cursor={{
                                            className: "stroke-utility-brand-600 stroke-2",
                                        }}
                                    />

                                    <Area
                                        isAnimationActive={false}
                                        className="text-utility-brand-600 [&_.recharts-area-area]:translate-y-[6px] [&_.recharts-area-area]:[clip-path:inset(0_0_6px_0)]"
                                        dataKey="A"
                                        name="Current period"
                                        type="linear"
                                        stroke="currentColor"
                                        strokeWidth={2}
                                        fill="none"
                                        activeDot={{
                                            className: "fill-bg-primary stroke-utility-brand-600 stroke-2",
                                        }}
                                    />

                                    <Area
                                        isAnimationActive={false}
                                        className="text-utility-brand-400 [&_.recharts-area-area]:translate-y-[6px] [&_.recharts-area-area]:[clip-path:inset(0_0_6px_0)]"
                                        dataKey="B"
                                        name="Previous period"
                                        type="linear"
                                        stroke="currentColor"
                                        strokeWidth={2}
                                        strokeDasharray="0.1 8"
                                        strokeLinecap="round"
                                        fill="none"
                                        activeDot={{
                                            className: "fill-bg-primary stroke-utility-brand-600 stroke-2",
                                        }}
                                    />
                                </AreaChart>
                            </ResponsiveContainer>
                        </div>
                    </div>

                    <div className="flex flex-col gap-5 px-4 lg:px-8">
                        <div className="flex flex-col items-start justify-between gap-4 lg:flex-row lg:items-center">
                            <p className="text-md font-semibold text-primary">Customers</p>
                            <div className="w-full lg:max-w-70">
                                <InputBase size="sm" type="search" shortcut aria-label="Search" placeholder="Search" icon={SearchLg} />
                            </div>
                        </div>

                        <div>
                            <Table
                                aria-label="Pages and screens"
                                selectionMode="multiple"
                                sortDescriptor={sortDescriptor}
                                onSortChange={setSortDescriptor}
                                className="bg-primary"
                            >
                                <Table.Header className="bg-primary [&_*:first-of-type]:pl-0!">
                                    <Table.Head id="name" label="Customer" isRowHeader allowsSorting className="w-full" />
                                    <Table.Head id="email" label="Email" allowsSorting className="max-lg:hidden" />
                                    <Table.Head id="date" label="Date" allowsSorting className="max-lg:hidden" />
                                    <Table.Head id="status" label="Status" allowsSorting className="max-lg:hidden" />
                                    <Table.Head id="amount" label="Amount" allowsSorting className="max-lg:hidden" />
                                    <Table.Head id="actions" />
                                </Table.Header>

                                <Table.Body items={sortedItems}>
                                    {(customer) => (
                                        <Table.Row id={customer.name} className="[&>*:first-of-type]:pl-0!">
                                            <Table.Cell className="text-nowrap">
                                                <div className="flex w-max items-center gap-3">
                                                    <Avatar src={customer.avatarUrl} initials={getInitials(customer.name)} alt={customer.name} size="md" />
                                                    <div>
                                                        <p className="text-sm font-medium text-primary">{customer.name}</p>
                                                        <p className="text-sm text-tertiary">{customer.username}</p>
                                                    </div>
                                                </div>
                                            </Table.Cell>
                                            <Table.Cell className="text-nowrap max-lg:hidden">{customer.email}</Table.Cell>
                                            <Table.Cell className="text-nowrap max-lg:hidden">{formatDate(customer.date)}</Table.Cell>
                                            <Table.Cell className="max-lg:hidden">
                                                <BadgeWithDot
                                                    color={customer.status === "paid" ? "success" : customer.status === "failed" ? "error" : "gray"}
                                                    type="modern"
                                                    size="sm"
                                                    className="capitalize"
                                                >
                                                    {customer.status}
                                                </BadgeWithDot>
                                            </Table.Cell>
                                            <Table.Cell className="max-lg:hidden">{formatCurrency(customer.amount)}</Table.Cell>

                                            <Table.Cell className="pr-0 pl-4">
                                                <div className="flex justify-end gap-0.5 max-lg:hidden">
                                                    <ButtonUtility size="xs" color="tertiary" tooltip="Delete" icon={Trash01} />
                                                    <ButtonUtility size="xs" color="tertiary" tooltip="Edit" icon={Edit01} />
                                                </div>
                                                <div className="flex items-center justify-end lg:hidden">
                                                    <DropdownIconSimple />
                                                </div>
                                            </Table.Cell>
                                        </Table.Row>
                                    )}
                                </Table.Body>
                            </Table>
                            <PaginationPageMinimalCenter page={1} total={10} />
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
};
