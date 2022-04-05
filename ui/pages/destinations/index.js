import Head from 'next/head'
import Link from 'next/link'

import Navigation from '../../components/nav/Navigation'

const Infrastructure = () => {
  return (
    <div>
      <Head>
        <title>Infra - Destinations</title>
      </Head>
      <Navigation />
      <>this is destinations page</>
      <Link href='/destinations/add/setup'>
        <a>add destination</a>
      </Link>
    </div>
  )
}

export default Infrastructure
