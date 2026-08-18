# Phase 04 - Secondary Adapters

## هدف
وصل کردن Core به بیرون (DB, JWT, Hashing, Generator) از طریق Port.

## چرا مهم است؟
اینجا محل dependency به کتابخانه‌هاست. باید قابل تعویض و قابل تست باشد.

## باید چه خروجی بدهی؟
- repository adapterهای کامل
- token manager و hasher adapter
- mapping خطاهای زیرساخت به خطاهای قابل فهم

## قدم‌های اجرایی
1. برای هر Port یک adapter concrete بنویس.
2. در repository، gorm/sql را محدود به همین پکیج نگه دار.
3. خطاهای `record not found` را map کن.
4. قراردادهای port را دقیق رعایت کن.

## اشتباهات رایج
- برگشت دادن errorهای raw دیتابیس به handler
- قراردادن منطق domain داخل repository
- اجرای business validation در adapter

## مسیر تحقیق
- Repository pattern in Go
- JWT best practices
- bcrypt trade-offs and cost
