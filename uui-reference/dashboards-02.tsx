"use client";

import { useMemo, useState } from "react";
import { Copy01, Edit01, Eye, Share01, SwitchHorizontal01, Trash01 } from "@untitledui/icons";
import type { SortDescriptor } from "react-aria-components";
import { HeaderNavigationBase } from "@/components/application/app-navigation/header-navigation";
import { FilterBarTabsAndSearchDemo } from "@/components/application/filter-bar/filter-bar.demo";
import { MetricsChart03 } from "@/components/application/metrics/metrics";
import { PaginationCardMinimal } from "@/components/application/pagination/pagination";
import { Table, TableCard } from "@/components/application/table/table";
import { TabList, Tabs } from "@/components/application/tabs/tabs";
import { BadgeWithDot } from "@/components/base/badges/badges";
import { Button } from "@/components/base/buttons/button";
import { ButtonUtility } from "@/components/base/buttons/button-utility";
import { Dropdown } from "@/components/base/dropdown/dropdown";
import { ProgressBar } from "@/components/base/progress-indicators/progress-indicators";

const tabs = [
    { id: "all", label: "All sessions" },
    { id: "direct", label: "Direct traffic" },
    { id: "organix", label: "Organic traffic" },
    { id: "paid", label: "Paid traffic" },
    { id: "mobile", label: "Mobile users" },
    { id: "returning", label: "Returning users" },
];

const pageData = [
    {
        page: "untitledui.com",
        sessions: 4288,
        avgTime: 84,
        percentageOfTotal: 62.4,
        folder: "General",
    },
    {
        page: "untitledui.com/free-icons",
        sessions: 582,
        avgTime: 68,
        percentageOfTotal: 8.2,
        folder: "General",
    },
    {
        page: "untitledui.com/icons",
        sessions: 464,
        avgTime: 72,
        percentageOfTotal: 7.6,
        folder: "General",
    },
    {
        page: "untitledui.com/components",
        sessions: 446,
        avgTime: 142,
        percentageOfTotal: 7.2,
        folder: "General",
    },
    {
        page: "untitledui.com/pricing",
        sessions: 382,
        avgTime: 48,
        percentageOfTotal: 7.0,
        folder: "General",
    },
    {
        page: "untitledui.com/faqs",
        sessions: 326,
        avgTime: 56,
        percentageOfTotal: 6.4,
        folder: "General",
    },
    {
        page: "untitledui.com/blog",
        sessions: 262,
        avgTime: 74,
        percentageOfTotal: 5.4,
        folder: "General",
    },
];

// Helper function for formatting seconds to "1m 24s" format
const formatSecondsToTime = (seconds: number): string => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;

    const minutesStr = minutes.toLocaleString("en-US");
    const secondsStr = remainingSeconds.toLocaleString("en-US");

    if (minutes > 0) {
        return remainingSeconds > 0 ? `${minutesStr}m ${secondsStr}s` : `${minutesStr}m`;
    }

    return `${secondsStr}s`;
};

export const Dashboard02 = () => {
    const [selectedTab, setSelectedTab] = useState<string>("all");
    const [sortDescriptor, setSortDescriptor] = useState<SortDescriptor>();

    const sortedItems = useMemo(() => {
        if (!sortDescriptor) return pageData;

        return pageData.toSorted((a, b) => {
            const first = a[sortDescriptor.column as keyof typeof a];
            const second = b[sortDescriptor.column as keyof typeof b];

            // Handle numbers (including avgTime which is now numeric)
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
        <div className="bg-primary">
            <HeaderNavigationBase
                activeUrl="/dashboard"
                items={[
                    { label: "Home", href: "/" },
                    { label: "Dashboard", href: "/dashboard" },
                    { label: "Projects", href: "/projects" },
                    { label: "Tasks", href: "/tasks" },
                    { label: "Reporting", href: "/reporting" },
                    { label: "Users", href: "/users" },
                ]}
            />

            <main className="bg-primary pt-8 pb-12 lg:pt-12 lg:pb-24">
                <div className="flex flex-col gap-8">
                    <div className="mx-auto flex w-full max-w-container flex-col gap-5 px-4 lg:px-8">
                        {/* Page header simple with search */}
                        <div className="relative flex flex-col gap-4">
                            <div className="flex flex-col gap-4 lg:flex-row">
                                <div className="flex flex-1 flex-col gap-0.5">
                                    <h1 className="text-xl font-semibold text-primary">Site traffic</h1>
                                </div>
                                <div className="flex items-start gap-3">
                                    <Button color="secondary" size="sm" iconLeading={SwitchHorizontal01}>
                                        Switch dashboard
                                    </Button>
                                    <Button color="secondary" size="sm">
                                        Export report
                                    </Button>
                                </div>
                            </div>
                        </div>

                        <div className="scrollbar-hide flex justify-between overflow-auto max-lg:-mx-4 max-lg:px-4">
                            <Tabs selectedKey={selectedTab} onSelectionChange={(value) => setSelectedTab(value as string)} className="w-auto">
                                <TabList type="button-minimal" items={tabs} />
                            </Tabs>

                            <Tabs defaultSelectedKey="30days" className="w-auto max-lg:hidden">
                                <TabList
                                    type="button-minimal"
                                    items={[
                                        {
                                            id: "12months",
                                            label: "12m",
                                        },
                                        {
                                            id: "30days",
                                            label: "30d",
                                        },
                                        {
                                            id: "7days",
                                            label: "7d",
                                        },
                                        {
                                            id: "24hours",
                                            label: "24h",
                                        },
                                    ]}
                                />
                            </Tabs>
                        </div>
                    </div>

                    <div className="mx-auto flex w-full max-w-container flex-col gap-5 px-4 md:flex-row md:flex-wrap lg:gap-6 lg:px-8">
                        <MetricsChart03
                            title="526"
                            subtitle="Total sessions"
                            className="flex-1 md:min-w-[320px]"
                            type="trend"
                            change="2.4%"
                            changeTrend="positive"
                            changeDescription="vs last month"
                            chartData={[{ value: 4 }, { value: 2 }, { value: 6 }, { value: 5 }, { value: 7 }]}
                        />
                        <MetricsChart03
                            title="2:24"
                            subtitle="Session duration"
                            className="flex-1 md:min-w-[320px]"
                            type="trend"
                            change="8.6%"
                            changeTrend="positive"
                            changeDescription="vs last month"
                            chartData={[{ value: 2 }, { value: 1 }, { value: 4 }, { value: 3 }, { value: 6 }, { value: 5 }, { value: 7 }]}
                        />
                        <MetricsChart03
                            title="316"
                            subtitle="Pages per session"
                            className="flex-1 md:min-w-[320px]"
                            type="trend"
                            change="6.0%"
                            changeTrend="positive"
                            changeDescription="vs last month"
                            chartData={[
                                { value: 2 },
                                { value: 2 },
                                { value: 1 },
                                { value: 2 },
                                { value: 3 },
                                { value: 3 },
                                { value: 4 },
                                { value: 6 },
                                { value: 5 },
                            ]}
                        />
                    </div>

                    <div className="mx-auto flex w-full max-w-container flex-col gap-6 px-4 lg:px-8">
                        <TableCard.Root className="-mx-4 rounded-none bg-secondary lg:mx-0 lg:rounded-xl">
                            <TableCard.Header
                                title="Pages and screens"
                                className="flex-row"
                                contentTrailing={
                                    <Dropdown.Root>
                                        <Dropdown.DotsButton />

                                        <Dropdown.Popover className="w-50">
                                            <Dropdown.Menu>
                                                <Dropdown.Item icon={Eye}>View more</Dropdown.Item>
                                                <Dropdown.Item icon={Share01}>Share</Dropdown.Item>
                                                <Dropdown.Item icon={Copy01}>Copy link</Dropdown.Item>
                                            </Dropdown.Menu>
                                        </Dropdown.Popover>
                                    </Dropdown.Root>
                                }
                            />

                            <div className="border-b border-secondary bg-primary px-4 py-3 lg:px-6">
                                <FilterBarTabsAndSearchDemo />
                            </div>

                            <Table
                                aria-label="Pages and screens"
                                selectionMode="multiple"
                                sortDescriptor={sortDescriptor}
                                onSortChange={setSortDescriptor}
                                className="bg-primary"
                            >
                                <Table.Header className="bg-primary">
                                    <Table.Head id="page" label="Page" allowsSorting isRowHeader className="w-full" />
                                    <Table.Head id="sessions" label="Sessions" allowsSorting />
                                    <Table.Head id="avgTime" label="Avg time" allowsSorting />
                                    <Table.Head id="percentageOfTotal" label="% of total" allowsSorting className="min-w-96" />
                                    <Table.Head id="folder" label="Folder" allowsSorting />
                                    <Table.Head id="actions" />
                                </Table.Header>

                                <Table.Body items={sortedItems}>
                                    {(stats) => (
                                        <Table.Row id={stats.page}>
                                            <Table.Cell className="font-medium!">{stats.page}</Table.Cell>
                                            <Table.Cell>{stats.sessions.toLocaleString()}</Table.Cell>
                                            <Table.Cell>{formatSecondsToTime(stats.avgTime)}</Table.Cell>
                                            <Table.Cell>
                                                <ProgressBar
                                                    labelPosition="right"
                                                    value={stats.percentageOfTotal}
                                                    valueFormatter={(value) => value.toFixed(1) + "%"}
                                                />
                                            </Table.Cell>
                                            <Table.Cell>
                                                <BadgeWithDot color="success" size="sm" type="modern">
                                                    {stats.folder}
                                                </BadgeWithDot>
                                            </Table.Cell>
                                            <Table.Cell className="px-4">
                                                <div className="flex justify-end gap-0.5">
                                                    <ButtonUtility size="xs" color="tertiary" tooltip="Delete" icon={Trash01} />
                                                    <ButtonUtility size="xs" color="tertiary" tooltip="Edit" icon={Edit01} />
                                                </div>
                                            </Table.Cell>
                                        </Table.Row>
                                    )}
                                </Table.Body>
                            </Table>
                            <PaginationCardMinimal align="right" page={1} total={10} className="bg-primary" />
                        </TableCard.Root>
                    </div>
                </div>
            </main>
        </div>
    );
};
