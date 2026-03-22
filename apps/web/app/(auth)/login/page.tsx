"use client";

import { Suspense } from "react";
import { LoginContent } from "./login-content";

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <LoginContent />
      </div>
    </Suspense>
  );
}
