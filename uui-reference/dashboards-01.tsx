"use client";

import { ArrowUp, ArrowUpRight, Bell01, Edit04, FilterLines, Settings01, UserPlus01 } from "@untitledui/icons";
import { Area, AreaChart, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer, XAxis } from "recharts";
import type { FeedItemType } from "@/components/application/activity-feed/activity-feed";
import { FeedItem } from "@/components/application/activity-feed/activity-feed";
import { NavButton } from "@/components/application/app-navigation/base-components/nav-button";
import { HeaderNavigationBase } from "@/components/application/app-navigation/header-navigation";
import { ChartTooltipContent } from "@/components/application/charts/charts-base";
import { DateRangePicker } from "@/components/application/date-picker/date-range-picker";
import { renderFilterRow, useFilterState } from "@/components/application/filter-bar/filter-bar.demo";
import { CountBadge, FilterDropdown } from "@/components/application/filter-bar/filter-dropdown-menu";
import { SectionHeader } from "@/components/application/section-headers/section-headers";
import { TabList, Tabs } from "@/components/application/tabs/tabs";
import type { BadgeColor } from "@/components/base/badges/badges";
import { BadgeWithIcon } from "@/components/base/badges/badges";
import { Button } from "@/components/base/buttons/button";
import { DropdownAvatar } from "@/components/base/dropdown/dropdown-avatar";
import { DropdownIconSimple } from "@/components/base/dropdown/dropdown-icon-simple";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";
import { cx } from "@/utils/cx";

const lineData = [
    {
        date: "2026-01-01",
        A: 600,
        B: 430,
    },
    {
        date: "2026-02-01",
        A: 620,
        B: 450,
    },
    {
        date: "2026-03-01",
        A: 630,
        B: 460,
    },
    {
        date: "2026-04-01",
        A: 650,
        B: 480,
    },
    {
        date: "2026-05-01",
        A: 600,
        B: 430,
    },
    {
        date: "2026-06-01",
        A: 650,
        B: 480,
    },
    {
        date: "2026-07-01",
        A: 620,
        B: 450,
    },
    {
        date: "2026-08-01",
        A: 750,
        B: 580,
    },
    {
        date: "2026-09-01",
        A: 780,
        B: 610,
    },
    {
        date: "2026-10-01",
        A: 750,
        B: 580,
    },
    {
        date: "2026-11-01",
        A: 780,
        B: 610,
    },
    {
        date: "2026-12-01",
        A: 820,
        B: 650,
    },
];

type Article = {
    id: string;
    href: string;
    thumbnailUrl: string;
    title: string;
    summary: string;
    category: {
        href: string;
        name: string;
    };
    author: {
        href: string;
        name: string;
        avatarUrl: string;
    };
    publishedAt: string;
    readingTime: string;
    tags: Array<{ name: string; color: BadgeColor<"color">; href: string }>;
    isFeatured?: boolean;
};

const articles: Article[] = [
    {
        id: "article-1",
        title: "Building your API Stack",
        summary: "The rise of RESTful APIs has been met by a rise in tools for creating, testing, and managing them.",
        href: "#",
        category: { name: "Design", href: "#" },
        thumbnailUrl: "https://www.untitledui.com/application/lumi-03.jpg",
        publishedAt: "18 Jan 2026",
        readingTime: "8 min read",
        author: {
            name: "Lana Steiner",
            href: "#",
            avatarUrl: "https://www.untitledui.com/images/avatars/lana-steiner?fm=webp&q=80",
        },
        tags: [
            { name: "Design", color: "brand", href: "#" },
            { name: "Research", color: "indigo", href: "#" },
            { name: "Presentation", color: "pink", href: "#" },
        ],
        isFeatured: true,
    },
    {
        id: "article-2",
        title: "Collaboration = better designer",
        summary: "Collaboration can make our teams stronger, and our individual designs better.",
        href: "#",
        category: { name: "Design", href: "#" },
        thumbnailUrl: "https://www.untitledui.com/application/lumi-04.jpg",

        publishedAt: "14 Jan 2026",
        readingTime: "8 min read",
        author: {
            name: "Natali Craig",
            href: "#",
            avatarUrl: "https://www.untitledui.com/images/avatars/natali-craig?fm=webp&q=80",
        },
        tags: [
            { name: "Product", color: "sky", href: "#" },
            { name: "Tools", color: "pink", href: "#" },
            { name: "SaaS", color: "pink", href: "#" },
        ],
    },
];

const feed: FeedItemType[] = [
    {
        id: "user-1",
        unseen: true,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/phoenix-baker?fm=webp&q=80",
            name: "Phoenix Baker",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Feb 2026" },
    },
    {
        id: "user-2",
        unseen: true,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/lana-steiner?fm=webp&q=80",
            name: "Lana Steiner",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Jan 2026" },
    },
    {
        id: "user-3",
        unseen: true,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/demi-wilkinson?fm=webp&q=80",
            name: "Demi Wilkinson",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Mar 2026" },
    },
    {
        id: "user-4",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/candice-wu?fm=webp&q=80",
            name: "Candice Wu",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Feb 2026" },
    },
    {
        id: "user-5",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/natali-craig?fm=webp&q=80",
            name: "Natali Craig",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Mar 2026" },
    },
    {
        id: "user-6",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/orlando-diggs?fm=webp&q=80",
            name: "Orlando Diggs",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Apr 2026" },
    },
    {
        id: "user-7",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/drew-cano?fm=webp&q=80",
            name: "Drew Cano",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Apr 2026" },
    },
    {
        id: "user-8",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/kate-morrison?fm=webp&q=80",
            name: "Kate Morrison",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Jan 2026" },
    },
    {
        id: "user-9",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/koray-okumus?fm=webp&q=80",
            name: "Koray Okumus",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Feb 2026" },
    },
    {
        id: "user-10",
        unseen: false,
        user: {
            avatarUrl: "https://www.untitledui.com/images/avatars/ava-wright?fm=webp&q=80",
            name: "Ava Wright",
            href: "#",
            status: "online",
        },
        action: { content: "Member since Mar 2026" },
    },
];

const Simple04Vertical = ({ article, imageClassName, className }: { article: Article; imageClassName?: string; className?: string }) => (
    <div className={cx("flex flex-col gap-4", className)}>
        <div className="relative">
            <a href={article.href} className="w-full" tabIndex={-1}>
                <img src={article.thumbnailUrl} alt={article.title} className={cx("aspect-[1.5] w-full object-cover", imageClassName)} />
            </a>
            <div className="absolute inset-x-0 bottom-0 overflow-hidden bg-linear-to-b from-transparent to-black/40">
                <div className="relative flex items-start justify-between bg-alpha-white/30 p-4 backdrop-blur-md before:absolute before:inset-x-0 before:top-0 before:h-px before:bg-alpha-white/30 md:p-5">
                    <div>
                        <a
                            href={article.author.href}
                            className="block rounded-xs text-sm font-semibold text-white outline-focus-ring focus-visible:outline-2 focus-visible:outline-offset-2"
                        >
                            {article.author.name}
                        </a>
                        <p className="text-sm text-white">{article.publishedAt}</p>
                    </div>
                    <a
                        href={article.category.href}
                        className="rounded-xs text-sm font-semibold text-white outline-focus-ring focus-visible:outline-2 focus-visible:outline-offset-2"
                    >
                        {article.category.name}
                    </a>
                </div>
            </div>
        </div>

        <div className="flex flex-col items-start gap-5">
            <div className="flex flex-col gap-1">
                <a
                    href={article.category.href}
                    className="flex justify-between gap-x-4 rounded-md text-lg font-semibold text-primary outline-focus-ring focus-visible:outline-2 focus-visible:outline-offset-2"
                >
                    {article.title}
                </a>
                <p className="line-clamp-2 text-md text-tertiary">{article.summary}</p>
            </div>

            <Button href={article.href} color="link-color" size="lg" iconTrailing={ArrowUpRight}>
                Read post
            </Button>
        </div>
    </div>
);

export const Dashboard01 = () => {
    const filterState = useFilterState();

    return (
        <div className="bg-primary">
            <HeaderNavigationBase
                activeUrl="/dashboard/overview"
                items={[
                    { label: "Home", href: "/" },
                    {
                        label: "Dashboard",
                        href: "/dashboard",
                        items: [
                            { label: "Overview", href: "/dashboard/overview" },
                            { label: "Notifications", href: "/dashboard/notifications" },
                            { label: "Analytics", href: "/dashboard/analytics" },
                            { label: "Saved reports", href: "/dashboard/saved-reports" },
                            { label: "Scheduled reports", href: "/dashboard/scheduled-reports" },
                            { label: "User reports", href: "/dashboard/user-reports" },
                        ],
                    },
                    { label: "Projects", href: "/projects" },
                    { label: "Tasks", href: "/tasks" },
                    { label: "Reporting", href: "/reporting" },
                    { label: "Users", href: "/users" },
                ]}
                actions={
                    <>
                        <div className="flex gap-0.5">
                            <NavButton icon={Settings01} label="Settings" href="/settings" tooltipPlacement="bottom" />
                            <div className="relative">
                                <NavButton icon={Bell01} label="Notifications" href="/notifications-01" tooltipPlacement="bottom" />
                                <div className="absolute -top-0.25 -right-0.25 flex size-3.5 items-center justify-center rounded-full bg-fg-error-primary text-[10px] font-bold text-white">
                                    2
                                </div>
                            </div>
                        </div>
                        <DropdownAvatar />
                    </>
                }
            />

            <main className="bg-primary pt-8 pb-12 lg:pt-12 lg:pb-24">
                <div className="flex flex-col gap-8">
                    <div className="flex flex-col gap-8">
                        <div className="mx-auto flex w-full max-w-container flex-col gap-5 px-4 lg:px-8">
                            {/* Page header simple with search */}
                            <div className="relative flex flex-col gap-4">
                                <div className="flex flex-col gap-4 lg:flex-row">
                                    <div className="flex flex-1 flex-col gap-0.5">
                                        <h1 className="text-xl font-semibold text-primary">Welcome back, Olivia</h1>
                                    </div>
                                </div>
                            </div>

                            <div className="flex gap-3 lg:justify-between">
                                <Tabs defaultSelectedKey="30days" className="w-auto">
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

                                <div className="flex gap-3">
                                    <DateRangePicker className="max-lg:hidden" />

                                    <FilterDropdown {...filterState} renderFilterRow={renderFilterRow}>
                                        <Button
                                            aria-label="Filters"
                                            color="secondary"
                                            size="sm"
                                            iconLeading={FilterLines}
                                            className={cx("max-h-9 max-md:hidden", filterState.appliedCount > 0 && "bg-primary_hover")}
                                        >
                                            <span className="flex items-center gap-1.5">
                                                Filters
                                                {filterState.appliedCount > 0 && <CountBadge count={filterState.appliedCount} />}
                                            </span>
                                        </Button>
                                    </FilterDropdown>
                                </div>
                            </div>
                        </div>

                        <div className="mx-auto flex w-full max-w-container flex-col gap-6 px-4 lg:flex-row lg:gap-8 lg:px-8">
                            <div className="flex flex-col gap-2">
                                <p className="text-sm font-medium text-tertiary">MRR</p>

                                <div className="flex items-center gap-4">
                                    <div className="flex items-start gap-0.5">
                                        <span className="pt-0.5 text-xl font-medium text-primary">$</span>
                                        <span className="text-display-md font-semibold text-primary">18,880</span>
                                    </div>

                                    <BadgeWithIcon type="modern" color="success" iconLeading={ArrowUp}>
                                        7.4%
                                    </BadgeWithIcon>
                                </div>
                            </div>

                            <div className="flex h-50 w-full flex-col gap-1.5 lg:h-60">
                                <ResponsiveContainer initialDimension={{ width: 1, height: 1 }} className="h-full">
                                    <AreaChart
                                        data={lineData}
                                        className="text-tertiary [&_.recharts-text]:text-xs"
                                        margin={{
                                            left: 5,
                                            right: 5,
                                        }}
                                    >
                                        <defs>
                                            <linearGradient id="gradient" x1="0" y1="0" x2="0" y2="1">
                                                <stop offset="0%" stopColor="currentColor" className="text-utility-neutral-500" stopOpacity="0.8" />
                                                <stop offset="80%" stopColor="currentColor" className="text-utility-neutral-500" stopOpacity="0" />
                                            </linearGradient>

                                            <pattern id="verticalLines" width="8" height="100%" fill="url(#gradient)" patternUnits="userSpaceOnUse">
                                                <line
                                                    x1="0"
                                                    y1="0"
                                                    x2="0"
                                                    y2="100%"
                                                    stroke="currentColor"
                                                    className="text-utility-neutral-200"
                                                    strokeWidth="1.5"
                                                />
                                                <rect width="100%" height="100%" fill="url(#gradient)" fillOpacity={0.15} />
                                            </pattern>
                                        </defs>

                                        <CartesianGrid vertical={false} stroke="currentColor" className="text-utility-neutral-100" />

                                        <XAxis
                                            fill="currentColor"
                                            axisLine={false}
                                            tickLine={false}
                                            tickMargin={10}
                                            interval="preserveStartEnd"
                                            dataKey="date"
                                            tickFormatter={(value) => new Date(value).toLocaleDateString(undefined, { month: "short" })}
                                        />

                                        <RechartsTooltip
                                            content={<ChartTooltipContent />}
                                            formatter={(value) =>
                                                value?.toLocaleString("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0 }) ?? ""
                                            }
                                            labelFormatter={(value) => new Date(value).toLocaleDateString(undefined, { month: "short", year: "numeric" })}
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
                                            fill="url(#verticalLines)"
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
                                            fill="none"
                                            activeDot={{
                                                className: "fill-bg-primary stroke-utility-brand-600 stroke-2",
                                            }}
                                        />
                                    </AreaChart>
                                </ResponsiveContainer>{" "}
                            </div>

                            <dl className="flex w-full max-w-60 flex-col gap-5">
                                <div className="flex flex-col gap-2">
                                    <dt className="text-sm font-medium text-tertiary">Total members</dt>
                                    <dd className="flex items-center gap-4">
                                        <span className="text-display-sm font-semibold text-primary">4,862</span>
                                        <BadgeWithIcon type="modern" color="success" iconLeading={ArrowUp}>
                                            9.2%
                                        </BadgeWithIcon>
                                    </dd>
                                </div>
                                <div className="flex flex-col gap-2">
                                    <dt className="text-sm font-medium text-tertiary">Paid members</dt>
                                    <dd className="flex items-center gap-4">
                                        <span className="text-display-sm font-semibold text-primary">2,671</span>
                                        <BadgeWithIcon type="modern" color="success" iconLeading={ArrowUp}>
                                            6.6%
                                        </BadgeWithIcon>
                                    </dd>
                                </div>
                                <div className="flex flex-col gap-2">
                                    <dt className="text-sm font-medium text-tertiary">Email open rate</dt>
                                    <dd className="flex items-center gap-4">
                                        <span className="text-display-sm font-semibold text-primary">82%</span>
                                        <BadgeWithIcon type="modern" color="success" iconLeading={ArrowUp}>
                                            8.1%
                                        </BadgeWithIcon>
                                    </dd>
                                </div>
                            </dl>
                        </div>

                        <div className="mx-auto flex w-full max-w-container flex-col gap-6 px-4 lg:px-8">
                            <SectionHeader.Root>
                                <SectionHeader.Group>
                                    <div className="flex flex-1 flex-col justify-center gap-0.5 self-stretch">
                                        <SectionHeader.Heading>Start creating content</SectionHeader.Heading>
                                    </div>

                                    <div className="absolute top-0 right-0 md:static">
                                        <DropdownIconSimple />
                                    </div>
                                </SectionHeader.Group>
                            </SectionHeader.Root>

                            <div className="flex flex-col gap-8 lg:flex-row">
                                <div className="flex flex-col gap-8">
                                    <div className="flex flex-col gap-5 md:flex-row md:flex-wrap lg:gap-6">
                                        <button className="flex flex-1 cursor-pointer gap-3 rounded-xl bg-primary p-4 shadow-xs ring-1 ring-secondary outline-focus-ring ring-inset focus-visible:outline-2 focus-visible:outline-offset-2 md:min-w-[320px]">
                                            <FeaturedIcon icon={UserPlus01} color="brand" theme="dark" size="lg" className="hidden lg:flex" />
                                            <FeaturedIcon icon={UserPlus01} color="brand" theme="dark" size="md" className="lg:hidden" />

                                            <div className="flex min-w-0 flex-1 flex-col items-start gap-0.5 text-left">
                                                <p className="text-md font-semibold text-secondary">Create your first member</p>
                                                <p className="max-w-full truncate text-sm text-tertiary">Add yourself or import from CSV</p>
                                            </div>
                                        </button>
                                        <button className="flex flex-1 cursor-pointer gap-3 rounded-xl bg-primary p-4 shadow-xs ring-1 ring-secondary outline-focus-ring ring-inset focus-visible:outline-2 focus-visible:outline-offset-2 md:min-w-[320px]">
                                            <FeaturedIcon icon={Edit04} color="brand" theme="dark" size="lg" className="hidden lg:flex" />
                                            <FeaturedIcon icon={Edit04} color="brand" theme="dark" size="md" className="lg:hidden" />

                                            <div className="flex min-w-0 flex-1 flex-col items-start gap-0.5 text-left">
                                                <p className="text-md font-semibold text-secondary">Create a new post</p>
                                                <p className="max-w-full truncate text-sm text-tertiary">Dive into the editor and start creating</p>
                                            </div>
                                        </button>
                                    </div>
                                    <div className="flex flex-col gap-6">
                                        <SectionHeader.Root>
                                            <SectionHeader.Group>
                                                <div className="flex flex-1 flex-col justify-center gap-0.5 self-stretch">
                                                    <SectionHeader.Heading>Recent posts</SectionHeader.Heading>
                                                </div>

                                                <div className="absolute top-0 right-0 md:static">
                                                    <DropdownIconSimple />
                                                </div>
                                            </SectionHeader.Group>
                                        </SectionHeader.Root>

                                        <div className="flex flex-col gap-6 md:flex-row md:flex-wrap">
                                            <Simple04Vertical article={articles[0]} className="flex-1 md:min-w-[320px]" />
                                            <Simple04Vertical article={articles[1]} className="flex-1 md:min-w-[320px]" />
                                        </div>
                                    </div>
                                </div>

                                <div className="flex w-full flex-col gap-6 lg:max-w-60">
                                    <p className="hidden text-sm font-medium text-secondary lg:block">Top members</p>

                                    <SectionHeader.Root className="lg:hidden">
                                        <SectionHeader.Group>
                                            <div className="flex flex-1 flex-col justify-center gap-0.5 self-stretch">
                                                <SectionHeader.Heading>Top members</SectionHeader.Heading>
                                            </div>

                                            <div className="absolute top-0 right-0 md:static">
                                                <DropdownIconSimple />
                                            </div>
                                        </SectionHeader.Group>
                                    </SectionHeader.Root>

                                    <ul className="flex flex-col gap-5">
                                        {feed.map((item) => (
                                            <li key={item.id}>
                                                <FeedItem {...item} size="sm" connector={false} />
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
};
