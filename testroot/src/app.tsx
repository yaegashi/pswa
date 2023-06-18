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
    path: <a href="/.auth/pswa/login">/.auth/pswa/login</a>,
    desc: "Azure AD sign in and redirect to the referer",
  },
  {
    key: 2,
    path: <a href="/.auth/pswa/login?return=/">/.auth/pswa/login?return=/</a>,
    desc: "Azure AD sign in and redirect to /",
  },
  {
    key: 3,
    path: <a href="/.auth/pswa/login?debug=true">/.auth/pswa/login?debug=true</a>,
    desc: "Azure AD sign in with debugging enabled",
  },
  {
    key: 4,
    path: <a href="/.auth/pswa/logout">/.auth/pswa/logout</a>,
    desc: "Sign out and redirect to the referer",
  },
  {
    key: 5,
    path: <a href="/.auth/pswa/logout?return=/">/.auth/pswa/logout?return=/</a>,
    desc: "Sign out and redirect to /",
  },
  {
    key: 6,
    path: <a href="/.auth/pswa/identity">/.auth/pswa/identity</a>,
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
  const ep = '/.auth/pswa/identity'
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
