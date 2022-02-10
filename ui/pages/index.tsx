import Link from "next/link";

export default function Index () {
  return (
    <div>
      You&apos;ve successfully installed Infra. To continue, see the <a className='underline' href="https://github.com/infrahq/infra">documentation</a>.
      <Link href='/account/login'>login</Link>
      <Link href='/account/register'>register</Link>
    </div>
  )
}
