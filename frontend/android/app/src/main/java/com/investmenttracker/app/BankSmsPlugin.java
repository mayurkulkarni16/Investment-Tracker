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

import java.util.Locale;
import java.util.regex.Pattern;

@CapacitorPlugin(name = "BankSms", permissions = {
    @Permission(alias = "sms", strings = {Manifest.permission.READ_SMS})
})
public class BankSmsPlugin extends Plugin {
    private static final Pattern SENDERS = Pattern.compile(
        "^(?:[A-Z0-9]{2}-)?(?:HDFCBK|HDFCBANK|ICICIB|ICICIBANK|AXISBK|AXISBANK|SBIINB|SBIBNK|SBICRD|SBICARD|SBIPSB|SBIUPI|ONECRD|ONECARD|ONCRD|YESBNK|YESBANK)(?:-[A-Z])?$"
    );
    private static final Pattern EXCLUDED = Pattern.compile(
        "\\b(?:OTP|one[- ]time password|verification code|offer|pre[- ]approved)\\b", Pattern.CASE_INSENSITIVE
    );

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
            long since = System.currentTimeMillis() - days * 86_400_000L;
            JSArray messages = new JSArray();
            int scanned = 0;
            boolean truncated = false;
            try (Cursor cursor = getContext().getContentResolver().query(
                Telephony.Sms.Inbox.CONTENT_URI,
                new String[]{Telephony.Sms.ADDRESS, Telephony.Sms.BODY, Telephony.Sms.DATE},
                Telephony.Sms.DATE + " >= ?", new String[]{Long.toString(since)},
                Telephony.Sms.DATE + " DESC"
            )) {
                if (cursor == null) {
                    call.reject("Android did not provide access to the SMS inbox.");
                    return;
                }
                while (cursor.moveToNext()) {
                    if (++scanned > 5000 || messages.length() >= 500) {
                        truncated = true;
                        break;
                    }
                    String sender = cursor.getString(0);
                    if (sender == null || !SENDERS.matcher(sender.toUpperCase(Locale.ROOT)).matches()) continue;
                    String body = cursor.getString(1);
                    if (body == null || EXCLUDED.matcher(body).find()) continue;
                    JSObject message = new JSObject();
                    message.put("sender", sender);
                    message.put("body", body);
                    message.put("receivedAt", cursor.getLong(2));
                    messages.put(message);
                }
                JSObject result = new JSObject();
                result.put("messages", messages);
                result.put("truncated", truncated);
                call.resolve(result);
            } catch (SecurityException error) {
                call.reject("SMS permission was revoked. Re-enable access or use manual entry.", error);
            } catch (SQLException | IllegalArgumentException | IllegalStateException error) {
                call.reject("Could not read the SMS inbox. Retry or use manual entry.", error);
            }
        });
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
