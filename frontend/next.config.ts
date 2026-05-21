import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    // Use environment variable if set, otherwise use Railway production URL
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ||
                   process.env.DATA_API_URL ||
                   "https://data-api-production-575e.up.railway.app";

    const ingestionUrl = process.env.NEXT_INGESTION_URL ||
                         "https://ingestion-production-6bec.up.railway.app";

    return {
      beforeFiles: [],
      afterFiles: [],
      fallback: [
        {
          source: "/api/:path*",
          destination: `${apiUrl}/api/:path*`,
        },
        {
          source: "/webhook/:path*",
          destination: `${ingestionUrl}/drop/:path*`,
        },
      ],
    };
  },
};

export default nextConfig;
