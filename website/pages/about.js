import Head from 'next/head'
import { useEffect, useState } from 'react'

import Layout from '../components/layout'

const members = [
  {
    name: 'Patrick Devine',
    role: 'CTO',
  },
  {
    name: 'Eva Ho',
    role: 'Engineering',
  },
  {
    name: 'Michael Chiang',
    role: 'Co-Founder',
  },
  {
    name: 'Elizabeth Kim',
    role: 'Engineering',
  },
  {
    name: 'Matt Williams',
    role: 'Evangelism',
  },
  {
    name: 'Mike Yang',
    role: 'Engineering',
  },
  {
    name: 'Steven Soroka',
    role: 'Engineering',
  },
  {
    name: 'Manning Fisher',
    role: 'Design',
  },
  {
    name: 'Daniel Nephin',
    role: 'Engineering',
  },
  {
    name: 'Bruce MacDonald',
    role: 'Engineering',
  },
  {
    name: 'Jeff Morgan',
    role: 'Co-Founder',
  },
]

const values = [
  {
    name: 'Obsess about the user',
    description:
      'We think deeply about how decisions affect the user. We craft an experience for the user that empowers and delights. We win when our users win.',
  },
  {
    name: 'Problem first',
    description:
      "We make sure we're solving problems that have large scale impact. We don't reinvent what already exists without merit. We leverage existing tools and products where we can. We take the time to think, be creative, and make sure we're solving the right problem.",
  },
  {
    name: 'Thoughtful & considerate',
    description:
      'We never treat anyone like a number and instead like a human whose opinions, feelings, and personal experience we care about. We respect and appreciate our peers both inside and outside the company.',
  },
  {
    name: 'Open to being wrong',
    description:
      "We understand we're not going to get everything right all the time, but it's only by being okay with making mistakes that we will be flexible enough to try different things to find the right answers. We aim to separate ourselves from our ideas, and seek first to understand others' perspectives. We make room for people to disagree with us, and are flexible enough to move on. Disagreement is important. It takes trust and courage to be able to move forward from disagreement.",
  },
  {
    name: 'Care about craftsmanship',
    description:
      'We obsess over the details that make a quality product. We care about our users having a tailored experience. We are especially drawn to simplicity as a core value of our design process. We strive to be the best at what we do, but we respect that this is a life-long journey to become masters of our craft, and we must embrace an attitude of continuous learning.',
  },
  {
    name: 'Challenge the status quo',
    description:
      'Everything can be improved, and we must challenge our assumptions. The best answer today may not be the best answer tomorrow, and we should watch out for opportunities to grow. Sometimes that means small improvements, and sometimes that means completely new ways of thinking.',
  },
  {
    name: 'Equality',
    description:
      "Everyone has something valuable to contribute, and we're all in this together. We don't hand out the grunt-work. We instead innovate to reduce that type of work, and own the rest ourselves. We want you to enjoy what you do, and work on things that give you energy, not sap it. To this end, we allow any team member to choose what specific area to work on; we trust them to make the right decision for themselves.",
  },
]

export default function About() {
  const [data, setData] = useState([])

  async function updatePostings() {
    const res = await fetch('https://api.lever.co/v0/postings/infra-hq')
    const data = await res.json()
    setData(data)
  }

  useEffect(() => {
    updatePostings()
  }, [])

  const teams = {}
  for (const d of data) {
    const team = d.categories.team
    if (!teams[team]) {
      teams[team] = []
    }

    teams[team] = [...teams[team], d]
  }

  return (
    <main className='mx-auto mb-32 max-w-screen-2xl space-y-40 px-8'>
      <Head>
        <title>Infra - About</title>
      </Head>
      <section className='flex h-screen flex-col'>
        <div className='flex max-w-4xl flex-1 items-center self-center'>
          <h1 className='md:leading-extra-tight mb-20 text-center text-4xl font-light tracking-tight md:text-5xl lg:text-6xl xl:text-8xl'>
            {`Our mission is to securely connect the world's infrastructure.`}
          </h1>
        </div>
      </section>

      {/* Intro */}
      <section className='mx-auto my-24 max-w-4xl tracking-tight'>
        <h3 className='text-2xl md:text-4xl md:leading-tight md:tracking-tight'>
          Hey there! We are the people building Infra.
          <br />
          <br />
          We&apos;re a group of builders and designers who love great design and
          believe in creating products that are <strong>simple</strong>,{' '}
          <strong>cohesive</strong>, <strong>familiar</strong> and{' '}
          <strong>polished</strong>.
          <br />
          <br />
          In our previous lives we worked hard to bring you a great experience
          through{' '}
          <strong>
            <a className='underline' href='https://docker.com'>
              Docker for Mac & Windows
            </a>
          </strong>
          ,{' '}
          <strong>
            <a className='underline' href='https://datadog.com'>
              Datadog
            </a>
          </strong>
          ,{' '}
          <strong>
            <a className='underline' href='https://dev.twitter.com'>
              Twitter
            </a>
          </strong>
          ,{' '}
          <strong>
            <a className='underline' href='https://consul.io'>
              Consul
            </a>
          </strong>
          ,{' '}
          <strong>
            <a
              className='underline'
              href='https://en.wikipedia.org/wiki/VMware_ESXi'
            >
              VMware ESXi
            </a>
          </strong>
          .
          <br />
          <br />
          These products are used daily by millions. They power billions of
          devices, apps and servers around the world.
          <br />
          <br />
          Most importantly, they&apos;ve become great businesses we&apos;re
          proud of.
        </h3>
      </section>

      {/* Team */}
      <section className='my-16 lg:my-32'>
        <h2 className='mb-12 text-center text-5xl tracking-tight md:text-6xl'>
          The team behind Infra
        </h2>
        <img
          alt='team'
          src='https://user-images.githubusercontent.com/251292/173832111-a31ab280-f615-45f8-8236-e90b1107ccac.png'
        />
        <div className='my-10 grid flex-none grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-3 md:gap-8 xl:grid-cols-4 2xl:grid-cols-5'>
          {members.map(t => (
            <div className='rounded-lg py-2 leading-tight md:py-0' key={t.name}>
              <h4 className='text-md md:text-xl md:leading-snug'>{t.name}</h4>
              <h5 className='text-md text-gray-400 md:text-lg md:leading-snug'>
                {t.role}
              </h5>
            </div>
          ))}
        </div>
      </section>

      {/* Values */}
      <section className='my-16 flex flex-col lg:my-24 lg:flex-row'>
        <h2 className='mb-12 mr-24 flex-1 text-5xl tracking-tight md:mb-24 md:text-6xl'>
          Our&nbsp;Values
        </h2>
        <div className='grid max-w-4xl grid-cols-1 gap-10 md:grid-cols-2 md:gap-8 lg:grid-cols-3 lg:gap-12'>
          {values.map(v => (
            <div key={v.name}>
              <h3 className='mb-2 text-xl'>{v.name}</h3>
              <p className='text-justify text-xl leading-tight text-gray-400'>
                {v.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* Work with us */}
      <section className='lg:my-54 my-16 flex flex-col text-center'>
        <h2 className='mb-6 flex-none text-5xl tracking-tight md:mb-12 md:text-6xl'>
          Work with us
        </h2>
        <a
          className='flex flex-1 flex-col text-xl text-gray-300 hover:underline'
          href='https://www.ycombinator.com/companies/infra/jobs'
        >
          See open roles ›
        </a>
      </section>
    </main>
  )
}

About.layout = function (page) {
  return <Layout>{page}</Layout>
}
