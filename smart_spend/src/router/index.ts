import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

// 定义路由元信息类型
declare module 'vue-router' {
    interface RouteMeta {
        requiresAuth?: boolean
        roles?: string[]
    }
}

const routes: Array<RouteRecordRaw> = [
    {
        path: '/',
        name: 'Login',
        component: () => import('../views/Login.vue'),
        meta: { requiresAuth: false }
    },
    {
        path: '/home',
        name: 'Home',
        component: () => import('../views/HomeView.vue'),
        meta: { requiresAuth: true }
    },
    {
        path: '/:pathMatch(.*)*',
        name: 'NotFound',
        component: () => import('../views/404.vue'),
        meta: {
            title: '404 Not Found'
        }
    }
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

// 路由守卫
router.beforeEach(async (to, from) => {
    const token = localStorage.getItem('token')

    // 检查路由是否需要认证
    if (to.meta.requiresAuth) {
        if (!token) {
            // 未登录，重定向到登录页
            return { name: 'Login', query: { redirect: to.fullPath } }
        }

        try {
            // 验证token有效性（可选）
            const isValid = await verifyToken(token)
            if (!isValid) {
                localStorage.removeItem('token')
                return { name: 'Login' }
            }
        } catch {
            return { name: 'Login' }
        }
    }

    // 如果已经登录但访问登录页，重定向到首页
    if (to.name === 'Login' && token) {
        return { name: 'Home' }
    }
})

// 验证token的函数（示例）
async function verifyToken(token: string): Promise<boolean> {
    // 这里实现token验证逻辑
    return true
}

export default router