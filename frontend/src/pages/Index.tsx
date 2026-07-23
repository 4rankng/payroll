import { Link } from "react-router-dom";
import { ArrowRight, ShieldCheck, WalletCards } from "lucide-react";

const Index = () => {
  return (
    <div className="min-h-screen bg-gradient-subtle text-foreground">
      <div className="mx-auto flex min-h-screen w-full max-w-5xl flex-col justify-center px-5 py-10 sm:px-8">
        <div className="mb-10 flex items-center gap-3">
          <img src="/logo-square.png" alt="TingTing" className="h-11 w-11 rounded-xl object-contain shadow-soft" />
          <div>
            <p className="font-display text-xl font-extrabold leading-tight">TingTing</p>
            <p className="text-sm text-muted-foreground">Quản lý lương công trình</p>
          </div>
        </div>

        <main className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
          <section className="space-y-6">
            <div className="inline-flex items-center gap-2 rounded-full border border-primary/10 bg-card px-3 py-1.5 text-xs font-semibold text-primary shadow-soft">
              <ShieldCheck className="h-3.5 w-3.5" />
              Bảo mật dữ liệu lương và ứng lương
            </div>
            <div className="space-y-4">
              <h1 className="font-display text-4xl font-extrabold leading-tight tracking-normal text-foreground sm:text-5xl">
                Bảng điều hành lương cho đội ngũ dự án
              </h1>
              <p className="max-w-2xl text-base leading-relaxed text-muted-foreground sm:text-lg">
                Theo dõi nhân sự, bảng công, ví lương và thanh toán trong một hệ thống vận hành rõ ràng cho admin, đối tác và nhân viên.
              </p>
            </div>

            <div className="flex flex-col gap-3 sm:flex-row">
              <Link
                to="/login"
                aria-label="Đăng nhập TingTing"
                className="inline-flex min-h-[44px] items-center justify-center gap-2 rounded-xl bg-primary px-5 py-3 text-sm font-bold text-primary-foreground shadow-card transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                Đăng nhập
                <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
          </section>

          <aside className="rounded-2xl border border-border/70 bg-card p-5 shadow-card">
            <div className="mb-5 flex items-center justify-between gap-3">
              <div>
                <p className="text-sm font-bold text-foreground">Trạng thái hệ thống</p>
                <p className="text-xs text-muted-foreground">Sẵn sàng xử lý ca làm và thanh toán</p>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-success/10 text-success">
                <WalletCards className="h-5 w-5" />
              </div>
            </div>
            <div className="grid gap-3">
              {[
                ["Bảng công", "Duyệt giờ làm theo dự án"],
                ["Ứng lương", "Theo dõi hạn mức và lịch sử"],
                ["Sổ cái", "Đối soát ví và giao dịch"],
              ].map(([title, description]) => (
                <div key={title} className="rounded-xl border border-border/60 bg-muted/20 p-4">
                  <p className="text-sm font-semibold text-foreground">{title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">{description}</p>
                </div>
              ))}
            </div>
          </aside>
        </main>
      </div>
    </div>
  );
};

export default Index;
