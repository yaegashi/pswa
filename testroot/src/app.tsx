import React, { ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom'
import { Layout, Typography, Breadcrumb, Alert, PageHeader, Table } from 'antd'

import './app.scss'

const { Header } = Layout
const { Title } = Typography;

const dataSource: { key: number, path: ReactNode, desc: string }[] = [
  {
    key: 0,
    path: <a href="/">/</a>,
    desc: "Top",
  },
  {
    key: 1,
    path: <a href="/.auth/login/aad">/.auth/login/aad</a>,
    desc: "Azure AD sign in",
  },
  {
    key: 2,
    path: <a href="/.auth/login/aad?return=/">/.auth/login/aad?return=/</a>,
    desc: "Azure AD sign in with redirect to /",
  },
  {
    key: 3,
    path: <a href="/.auth/login/aad?debug=true">/.auth/login/aad?debug=true</a>,
    desc: "Azure AD sign in with debug",
  },
  {
    key: 4,
    path: <a href="/.auth/logout">/.auth/logout</a>,
    desc: "Sign out",
  },
  {
    key: 5,
    path: <a href="/.auth/logout?return=/">/.auth/logout?return=/</a>,
    desc: "Sign out with redirect to /",
  },
  {
    key: 6,
    path: <a href="/.auth/me">/.auth/me</a>,
    desc: "Authentication status",
  },
]

const columns = [
  {
    title: 'Location path',
    dataIndex: 'path',
    key: 'path'
  },
  {
    title: 'Description',
    dataIndex: 'desc',
    key: 'desc'
  },
]

const AppBreadcrumb: React.FC = () => {
  const location = useLocation()
  const segs = location.pathname.split('/').filter(i => i)
  const breadcrumbItems: ReactNode[] = [
    <Breadcrumb.Item key={-3}>Location</Breadcrumb.Item>,
    <Breadcrumb.Separator key={-2}>:</Breadcrumb.Separator>,
    <Breadcrumb.Item key={-1}><Link to="/">root</Link></Breadcrumb.Item>
  ]
  for (var i = 0; i < segs.length; i++) {
    breadcrumbItems.push(
      <Breadcrumb.Separator key={i * 2}>/</Breadcrumb.Separator>
    )
    breadcrumbItems.push(
      <Breadcrumb.Item key={i * 2 + 1}>
        <Link to={segs.slice(0, i + 1).join('/')}>{segs[i]}</Link>
      </Breadcrumb.Item>
    )
  }
  return <Breadcrumb separator="">{breadcrumbItems}</Breadcrumb>
}

const AppLocation: React.FC = () => {
  const location = useLocation()
  return (
    <Alert
      message="You're seeing a forbidden location!"
      description={
        <span>
          Location <b>{location.pathname}</b> should be handled by the PSWA server.
          The authentication configuration is not completed.
        </span>
      }
      type="error"
      className="app-alert"
    />
  )
}

const AppAuth: React.FC = () => {
  const ep = '/.auth/me'
  const [alertMsg, setAlertMsg] = React.useState(`GET ${ep} ...`)
  const [alertType, setAlertType] = React.useState<'info' | 'error'>('info')
  React.useEffect(
    () => {
      (async () => {
        try {
          const res = await fetch(ep)
          if (!res.ok) {
            throw new Error(`GET ${ep}: fetch failed: ${res.status} ${res.statusText}: ${await res.text()}`)
          }
          const contentType = res.headers.get('Content-Type')
          if (!contentType || contentType.indexOf('application/json') === -1) {
            throw new Error(`GET ${ep}: invalid content type: "${contentType}"`)
          }
          setAlertMsg(JSON.stringify(await res.json()))
          setAlertType('info')
        } catch (err: unknown) {
          setAlertMsg(err.toString())
          setAlertType('error')
        }
      })()
    },
    []
  );
  return <Alert message={alertMsg} type={alertType} className="app-alert" />
}

const AppRoot: React.FC = () => {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header>
        <Link to="/" style={{ float: 'left', fontSize: '20px', color: 'white' }}>
          PSWA: Protected Static Web App
        </Link>
      </Header>
      <PageHeader title="PSWA Landing Page" breadcrumbRender={() => <AppBreadcrumb />}>
        <Title level={5}>Authentication status</Title>
        <AppAuth />
        <Routes>
          <Route path="/.auth/*" element={<AppLocation />}></Route>
        </Routes>
        <Title level={5}>Navigation links</Title>
        <Table dataSource={dataSource} columns={columns} pagination={false} />
      </PageHeader>
    </Layout>
  )
}

export const App: React.FC = () => <BrowserRouter><AppRoot /></BrowserRouter>
