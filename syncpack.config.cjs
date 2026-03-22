/** @type {import('syncpack').RcFile} */
module.exports = {
  // Version groups to keep related packages in sync
  versionGroups: [
    {
      label: "React ecosystem",
      dependencies: ["react", "react-dom", "@types/react", "@types/react-dom"],
      pinVersion: "latest",
    },
    {
      label: "TanStack",
      dependencies: ["@tanstack/react-query"],
      pinVersion: "latest",
    },
    {
      label: "Tailwind CSS",
      dependencies: [
        "tailwindcss",
        "@tailwindcss/postcss",
        "autoprefixer",
        "postcss",
      ],
      pinVersion: "latest",
    },
    {
      label: "Radix UI",
      dependencies: ["radix-ui"],
      pinVersion: "latest",
    },
    {
      label: "Lucide Icons",
      dependencies: ["lucide-react"],
      pinVersion: "latest",
    },
    {
      label: "Recharts",
      dependencies: ["recharts"],
      pinVersion: "latest",
    },
    {
      label: "TypeScript",
      dependencies: ["typescript"],
      pinVersion: "latest",
    },
    {
      label: "@types/node",
      dependencies: ["@types/node"],
      pinVersion: "latest",
    },
    {
      label: "ESLint ecosystem",
      dependencies: [
        "eslint",
        "@typescript-eslint/*",
        "eslint-plugin-*",
        "@eslint/*",
      ],
      pinVersion: "latest",
    },
    {
      label: "Connect RPC",
      dependencies: [
        "@bufbuild/protobuf",
        "@connectrpc/connect",
        "@connectrpc/connect-web",
        "@bufbuild/protoc-gen-es",
      ],
      pinVersion: "latest",
    },
    {
      label: "Uppy",
      dependencies: ["@uppy/*"],
      pinVersion: "latest",
    },
    {
      label: "Playwright",
      dependencies: ["@playwright/test", "playwright"],
      pinVersion: "latest",
    },
  ],
};
