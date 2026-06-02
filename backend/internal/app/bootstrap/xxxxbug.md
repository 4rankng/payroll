I am facing a bug where transactions for weekly payment schedules are not full settled

How to reproduce:
0. Run make restore at payroll/backend to reset local dev db to prod state
1. Export sao ke for weekly paymetn schedule on 26 May, 2 June, 26 June (can store in ~/Downloads/ folder)
(when you call curl to export sao ke , just neeed to put date 2026-05-26, 2026-06-02, 2026-06-26)
2. Settle these sao ke files
3. Check if all timesheets are settled
4. Check if all related transactions are settled

To call the curl from API server, you can log in with
username: frankng
password: Admin123

To check db, here is the format
docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "SELECT id, total_amount, settled_amount,status FROM transactions WHERE id = 83;"

Your goal is find the root cause of the problem and propose a way to fix them
