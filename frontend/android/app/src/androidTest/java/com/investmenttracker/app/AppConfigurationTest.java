package com.investmenttracker.app;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import android.Manifest;
import android.content.Context;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import com.getcapacitor.JSObject;
import java.util.Arrays;
import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.Test;
import org.junit.runner.RunWith;

@RunWith(AndroidJUnit4.class)
public class AppConfigurationTest {
    @Test
    public void financialAppDisablesBackupAndDeclaresOnlyReadSms() throws Exception {
        Context context = InstrumentationRegistry.getInstrumentation().getTargetContext();
        assertEquals("com.investmenttracker.app", context.getPackageName());
        PackageInfo info = context.getPackageManager().getPackageInfo(
            context.getPackageName(), PackageManager.GET_PERMISSIONS
        );
        assertTrue(Arrays.asList(info.requestedPermissions).contains(Manifest.permission.READ_SMS));
        assertFalse(Arrays.asList(info.requestedPermissions).contains(Manifest.permission.SEND_SMS));
        assertFalse((info.applicationInfo.flags & ApplicationInfo.FLAG_ALLOW_BACKUP) != 0);
    }

    @Test
    public void scanTraversesBusyWindowIncludingTiedTimestampsAndUnknownSenders() throws Exception {
        try (SQLiteDatabase db = SQLiteDatabase.create(null)) {
            db.execSQL("CREATE TABLE sms (_id INTEGER PRIMARY KEY, address TEXT, body TEXT, date INTEGER)");
            db.execSQL("INSERT INTO sms VALUES (1, 'OLD', 'outside window', 199)");
            for (int i = 2; i <= 5202; i++) {
                db.execSQL("INSERT INTO sms VALUES (?, ?, ?, ?)", new Object[]{
                    i, i == 3 ? null : "VM-NEWBNK-S", "alert-" + i, i == 2 ? 200 : 300
                });
            }
            db.execSQL("INSERT INTO sms VALUES (5203, 'NEW', 'outside frozen window', 301)");

            JSONObject next = null;
            int expectedId = 5202;
            int scanned = 0;
            do {
                String[] args = next == null ? new String[]{"200", "300"} : new String[]{
                    "200", next.getString("until"), next.getString("date"), next.getString("date"), next.getString("id")
                };
                try (Cursor cursor = db.query("sms", new String[]{"address", "body", "date", "_id"},
                    BankSmsPlugin.WINDOW + (next == null ? "" : BankSmsPlugin.AFTER), args,
                    null, null, BankSmsPlugin.ORDER)) {
                    JSObject page = BankSmsPlugin.readPage(cursor, 300);
                    scanned += page.getInt("scanned");
                    JSONArray messages = page.getJSONArray("messages");
                    for (int i = 0; i < messages.length(); i++) {
                        if (expectedId == 3) expectedId--; // The missing sender is omitted, not skipped by the cursor.
                        JSONObject message = messages.getJSONObject(i);
                        assertEquals("VM-NEWBNK-S", message.getString("sender"));
                        assertEquals("alert-" + expectedId--, message.getString("body"));
                    }
                    next = page.optJSONObject("nextCursor");
                }
            } while (next != null);
            assertEquals(5201, scanned);
            assertEquals(1, expectedId);
        }
    }
}
