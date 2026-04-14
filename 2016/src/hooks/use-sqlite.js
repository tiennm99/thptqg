import { useState, useEffect, useRef } from "react";
import initSqlJs from "sql.js";

const SQL_WASM_URL = "https://sql.js.org/dist/sql-wasm.wasm";

/**
 * Hook to load a SQLite database from a gzipped URL into sql.js.
 * Returns { db, loading, error, progress }.
 */
export function useSqlite(dbUrl) {
  const [db, setDb] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [progress, setProgress] = useState(0);
  const dbRef = useRef(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const SQL = await initSqlJs({ locateFile: () => SQL_WASM_URL });

        const response = await fetch(dbUrl);
        if (!response.ok)
          throw new Error(`Failed to fetch database: ${response.status}`);

        const contentLength = +response.headers.get("Content-Length") || 0;
        const reader = response.body.getReader();
        const chunks = [];
        let received = 0;

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          chunks.push(value);
          received += value.length;
          if (contentLength > 0) {
            setProgress(Math.round((received / contentLength) * 100));
          }
        }

        if (cancelled) return;

        const blob = new Blob(chunks);
        let arrayBuffer;

        if (dbUrl.endsWith(".gz")) {
          const ds = new DecompressionStream("gzip");
          const decompressed = blob.stream().pipeThrough(ds);
          const decompressedBlob = await new Response(decompressed).blob();
          arrayBuffer = await decompressedBlob.arrayBuffer();
        } else {
          arrayBuffer = await blob.arrayBuffer();
        }

        if (cancelled) return;

        const database = new SQL.Database(new Uint8Array(arrayBuffer));
        dbRef.current = database;
        setDb(database);
        setLoading(false);
      } catch (err) {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      }
    }

    load();

    return () => {
      cancelled = true;
      if (dbRef.current) {
        dbRef.current.close();
        dbRef.current = null;
      }
    };
  }, [dbUrl]);

  return { db, loading, error, progress };
}
