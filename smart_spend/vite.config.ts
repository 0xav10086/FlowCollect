import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { resolve } from 'path'  // 使用具名导入

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        vue(),
        tailwindcss(),
        AutoImport({
            // 自动导入 Vue 相关函数，如：ref, reactive, toRef...
            imports: ['vue', 'vue-router', 'pinia'],
            // 自动导入 Element Plus 相关函数，如：ElMessage, ElMessageBox...
            resolvers: [ElementPlusResolver()],
            dts: 'src/auto-imports.d.ts', // 生成自动导入的 TypeScript 声明
            eslintrc: {
                enabled: true, // 启用 eslint 规则生成
            },
        }),
        Components({
            resolvers: [
                // 自动导入 Element Plus 组件
                ElementPlusResolver(),
            ],
            dts: 'src/components.d.ts', // 生成组件 TypeScript 声明
        }),
    ],

    // 解析配置
    resolve: {
        alias: {
            '@': resolve(__dirname, 'src'), // 路径别名
            '@components': resolve(__dirname, 'src/components'),
            '@views': resolve(__dirname, 'src/views'),
            '@utils': resolve(__dirname, 'src/utils'),
            '@api': resolve(__dirname, 'src/api'),
        },
        extensions: ['.js', '.ts', '.jsx', '.tsx', '.json', '.vue'] // 省略的扩展名
    },

    // 开发服务器配置
    server: {
        port: 8687,
        open: true, // 自动打开浏览器
        host: true, // 监听所有地址
        proxy: {
            '/api': {
                target: 'http://localhost:8686',
                changeOrigin: true,
            },
        },
    },

    // 构建配置
    build: {
        outDir: 'dist',
        sourcemap: false,
        rollupOptions: {
            output: {
                chunkFileNames: 'js/[name]-[hash].js',
                entryFileNames: 'js/[name]-[hash].js',
                assetFileNames: '[ext]/[name]-[hash].[ext]',
            },
        },
    },
})