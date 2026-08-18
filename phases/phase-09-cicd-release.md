# Phase 09 - CI/CD و Release

## هدف
تحویل تکرارپذیر، امن، و قابل rollback.

## چرا مهم است؟
پروژه‌ای که deploy مطمئن ندارد در عمل قابل اتکا نیست.

## باید چه خروجی بدهی؟
- pipeline CI سبز و پایدار
- release process مستند
- rollback plan تست‌شده

## قدم‌های اجرایی
1. CI: lint + test + build.
2. image versioned بساز (semantic tag یا commit sha).
3. migration strategy قبل/حین deploy مشخص کن.
4. health-gated deployment اجرا کن.
5. rollback را تمرین کن.

## اشتباهات رایج
- deploy بدون migration plan
- نبود rollback واقعی
- عدم همخوانی نسخه image و کد

## مسیر تحقیق
- Trunk-based development
- Database migration safety
- Blue/Green or Rolling deployment basics
