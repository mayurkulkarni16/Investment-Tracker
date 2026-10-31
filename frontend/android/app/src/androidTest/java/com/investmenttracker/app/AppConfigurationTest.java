package com.investmenttracker.app;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import android.Manifest;
import android.content.Context;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import java.util.Arrays;
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
}
