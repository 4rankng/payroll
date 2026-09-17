import { AlertCircle, FileText, Mail, Paperclip, Send, X } from 'lucide-react';
import { PageHeader } from '@/components/shared/PageHeader';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Textarea } from '@/components/ui/textarea';
import { useEmailComposer } from '@/hooks/useEmailComposer';

const recipientLabel = (name: string | undefined, address: string) => name ? `${name} <${address}>` : address;

interface AdminEmailComposerProps {
  embedded?: boolean;
}

export function AdminEmailComposer({ embedded = false }: AdminEmailComposerProps) {
  const composer = useEmailComposer();

  return (
    <div className={embedded ? 'space-y-5' : 'p-4 lg:p-6 max-w-[1320px] mx-auto space-y-5 pb-[calc(5rem+env(safe-area-inset-bottom))]'}>
      {!embedded && <PageHeader icon={Mail} title="Gửi email" description="Soạn email HTML, chọn địa chỉ gửi đã xác minh và xem trước trước khi gửi." />}

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.8fr)]">
        <section className="min-w-0 space-y-4 rounded-xl border bg-card p-4 sm:p-5" aria-labelledby="email-composer-title">
          <h2 id="email-composer-title" className="text-base font-semibold">Thông tin email</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="email-from">Địa chỉ gửi</Label>
              <Select value={composer.from} onValueChange={composer.setFrom} disabled={composer.isLoadingSenders || composer.sendEmail.isPending}>
                <SelectTrigger id="email-from" aria-label="Địa chỉ gửi"><SelectValue placeholder="Chọn địa chỉ gửi" /></SelectTrigger>
                <SelectContent>{composer.senders.map((sender) => <SelectItem key={sender.address} value={sender.address}>{recipientLabel(sender.name, sender.address)}</SelectItem>)}</SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">Chỉ hiện các địa chỉ đã được cấu hình và xác minh.</p>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="email-reply-to">Nhận phản hồi tại (tuỳ chọn)</Label>
              <Input id="email-reply-to" type="email" value={composer.replyTo} onChange={(event) => composer.setReplyTo(event.target.value)} placeholder="hotro@tingting.vip" disabled={composer.sendEmail.isPending} />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="email-to">Người nhận</Label>
            <Input id="email-to" type="text" value={composer.recipients} onChange={(event) => { composer.setRecipients(event.target.value); composer.setError(''); }} placeholder="email1@congty.vn, email2@congty.vn" disabled={composer.sendEmail.isPending} />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5"><Label htmlFor="email-cc">CC (tuỳ chọn)</Label><Input id="email-cc" value={composer.cc} onChange={(event) => composer.setCc(event.target.value)} placeholder="email@congty.vn" disabled={composer.sendEmail.isPending} /></div>
            <div className="space-y-1.5"><Label htmlFor="email-bcc">BCC (tuỳ chọn)</Label><Input id="email-bcc" value={composer.bcc} onChange={(event) => composer.setBcc(event.target.value)} placeholder="email@congty.vn" disabled={composer.sendEmail.isPending} /></div>
          </div>
          <div className="space-y-1.5"><Label htmlFor="email-subject">Tiêu đề</Label><Input id="email-subject" value={composer.subject} onChange={(event) => { composer.setSubject(event.target.value); composer.setError(''); }} placeholder="Nhập tiêu đề email" disabled={composer.sendEmail.isPending} /></div>

          <Tabs value={composer.mode} onValueChange={composer.setMode} className="space-y-3">
            <TabsList className="grid h-auto w-full grid-cols-2" aria-label="Chế độ soạn email"><TabsTrigger className="min-w-0 whitespace-normal text-center" value="rich">Soạn / dán định dạng</TabsTrigger><TabsTrigger className="min-w-0 whitespace-normal text-center" value="html">Dán mã HTML</TabsTrigger></TabsList>
            <TabsContent value="rich" className="space-y-2">
              <Label htmlFor="email-rich-editor">Nội dung</Label>
              <div id="email-rich-editor" ref={composer.editorRef} contentEditable suppressContentEditableWarning onInput={composer.handleEditorInput} aria-label="Nội dung email có định dạng" role="textbox" aria-multiline="true" tabIndex={0} className="min-h-[280px] overflow-x-auto rounded-md border bg-background px-3 py-2 text-sm leading-6 outline-none focus-visible:ring-2 focus-visible:ring-ring" data-placeholder="Dán nội dung đã định dạng từ Gmail, Google Docs hoặc trình soạn thảo khác..." />
              <p className="text-xs text-muted-foreground">Dán nội dung có định dạng để giữ bảng, liên kết, danh sách và kiểu chữ inline tương thích Gmail.</p>
            </TabsContent>
            <TabsContent value="html" className="space-y-2">
              <Label htmlFor="email-html-source">Mã HTML</Label>
              <Textarea id="email-html-source" value={composer.sourceHtml} onChange={(event) => { composer.setSourceHtml(event.target.value); composer.setError(''); }} rows={13} placeholder="Dán HTML đã soạn (ưu tiên table và inline CSS để tương thích Gmail)..." className="font-mono text-xs leading-5" disabled={composer.sendEmail.isPending} />
            </TabsContent>
          </Tabs>

          <div className="space-y-2">
            <Label htmlFor="email-attachments">Tệp đính kèm (tuỳ chọn)</Label>
            <Input id="email-attachments" type="file" multiple accept=".pdf,.xlsx,.xls,.doc,.docx,.png,.jpg,.jpeg,.gif,.txt,.csv" onChange={composer.handleAttachmentChange} disabled={composer.sendEmail.isPending} />
            {composer.attachments.length > 0 && <div className="flex flex-wrap gap-2">{composer.attachments.map((file) => <Badge key={`${file.name}-${file.size}`} variant="secondary" className="min-w-0 max-w-full gap-1"><FileText className="h-3 w-3 shrink-0" /><span className="truncate" title={file.name}>{file.name}</span><button type="button" className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={() => composer.setAttachments((current) => current.filter((item) => item !== file))} aria-label={`Bỏ tệp ${file.name}`}><X className="h-3 w-3" /></button></Badge>)}</div>}
          </div>
          {composer.error && <p role="alert" className="flex items-center gap-1.5 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"><AlertCircle className="h-4 w-4" />{composer.error}</p>}
          <Button type="button" className="w-full min-h-11" onClick={composer.handleSend} disabled={!composer.canSend || composer.sendEmail.isPending}><Send className="mr-2 h-4 w-4" />{composer.sendEmail.isPending ? 'Đang gửi email...' : 'Gửi email'}</Button>
        </section>

        <aside className="min-w-0 space-y-3 xl:sticky xl:top-4 xl:self-start" aria-labelledby="email-preview-title">
          <div><h2 id="email-preview-title" className="text-base font-semibold">Xem trước</h2><p className="text-xs text-muted-foreground">Banner TingTing được máy chủ tự động chèn khi gửi.</p></div>
          <iframe title="Bản xem trước email" sandbox="" srcDoc={composer.previewDocument} className="h-[620px] w-full rounded-xl border bg-white" />
          <div className="flex items-start gap-2 rounded-lg border border-muted bg-muted/40 p-3 text-xs text-muted-foreground"><Paperclip className="mt-0.5 h-4 w-4 shrink-0" />Gmail hiển thị tốt nhất với bảng và CSS inline. JavaScript và nội dung không an toàn không được chạy trong bản xem trước.</div>
        </aside>
      </div>
    </div>
  );
}
