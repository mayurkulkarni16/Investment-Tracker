package com.investmenttracker.app;

import android.Manifest;
import android.content.Intent;
import android.content.ActivityNotFoundException;
import android.database.Cursor;
import android.database.SQLException;
import android.net.Uri;
import android.provider.Settings;
import android.provider.Telephony;

import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

@CapacitorPlugin(name = "BankSms", permissions = {
    @Permission(alias = "sms", strings = {Manifest.permission.READ_SMS})
})
public class BankSmsPlugin extends Plugin {
    static final int PAGE_SIZE = 100;
    static final String WINDOW = Telephony.Sms.DATE + " >= ? AND " + Telephony.Sms.DATE + " <= ?";
    static final String AFTER = " AND (" + Telephony.Sms.DATE + " < ? OR (" + Telephony.Sms.DATE
        + " = ? AND " + Telephony.Sms._ID + " < ?))";
    static final String ORDER = Telephony.Sms.DATE + " DESC, " + Telephony.Sms._ID + " DESC";

    @PluginMethod
    public void scan(PluginCall call) {
        int days = call.getInt("days", 7);
        if (days < 1 || days > 90) {
            call.reject("Choose a scan window between 1 and 90 days.");
            return;
        }
        if (getPermissionState("sms") != PermissionState.GRANTED) {
            requestPermissionForAlias("sms", call, "smsPermissionResult");
            return;
        }
        readInbox(call);
    }

    @PermissionCallback
    private void smsPermissionResult(PluginCall call) {
        if (getPermissionState("sms") != PermissionState.GRANTED) {
            call.reject("SMS access was denied. You can paste a message manually or enable SMS access in Android app settings.");
            return;
        }
        readInbox(call);
    }

    private void readInbox(PluginCall call) {
        getBridge().execute(() -> {
            int days = call.getInt("days", 7);
            JSObject previous = call.getObject("cursor");
            long now = System.currentTimeMillis();
            long end = previous == null ? now : previous.optLong("until", -1);
            long afterDate = previous == null ? 0 : previous.optLong("date", -1);
            long afterId = previous == null ? 0 : previous.optLong("id", -1);
            if (end < 0 || end > now || afterDate < 0 || afterDate > end || afterId < 0) {
                call.reject("Invalid SMS scan cursor. Start a new scan.");
                return;
            }
            long since = end - days * 86_400_000L;
            Long sinceMs = call.getLong("sinceMs");
            if (sinceMs != null) since = Math.max(sinceMs, since);
            String start = Long.toString(since);
            String until = Long.toString(end);
            String[] args = previous == null ? new String[]{start, until} : new String[]{
                start, until, Long.toString(afterDate), Long.toString(afterDate), Long.toString(afterId)
            };
            try (Cursor cursor = getContext().getContentResolver().query(
                Telephony.Sms.Inbox.CONTENT_URI,
                new String[]{Telephony.Sms.ADDRESS, Telephony.Sms.BODY, Telephony.Sms.DATE, Telephony.Sms._ID},
                WINDOW + (previous == null ? "" : AFTER), args, ORDER
            )) {
                if (cursor == null) {
                    call.reject("Android did not provide access to the SMS inbox.");
                    return;
                }
                call.resolve(readPage(cursor, end));
            } catch (SecurityException error) {
                call.reject("SMS permission was revoked. Re-enable access or use manual entry.", error);
            } catch (SQLException | IllegalArgumentException | IllegalStateException error) {
                call.reject("Could not read the SMS inbox. Retry or use manual entry.", error);
            }
        });
    }

    static JSObject readPage(Cursor cursor, long until) {
        JSArray messages = new JSArray();
        int scanned = 0;
        long date = 0;
        long id = 0;
        while (scanned < PAGE_SIZE && cursor.moveToNext()) {
            scanned++;
            date = cursor.getLong(2);
            id = cursor.getLong(3);
            String sender = cursor.getString(0);
            String body = cursor.getString(1);
            if (sender == null || body == null || body.length() > 5000) continue;
            JSObject message = new JSObject();
            message.put("sender", sender);
            message.put("body", body);
            message.put("receivedAt", date);
            messages.put(message);
        }
        JSObject result = new JSObject();
        result.put("messages", messages);
        result.put("scanned", scanned);
        if (scanned == PAGE_SIZE && cursor.moveToNext()) {
            result.put("nextCursor", new JSObject().put("until", until).put("date", date).put("id", id));
        }
        return result;
    }

    @PluginMethod
    public void openSettings(PluginCall call) {
        try {
            Intent intent = new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                Uri.parse("package:" + getContext().getPackageName()));
            getActivity().startActivity(intent);
            call.resolve();
        } catch (ActivityNotFoundException error) {
            call.reject("Could not open Android settings. Open the app's permissions from system settings.", error);
        }
    }
}
