import { useRouter } from "next/router";
import useSWR from "swr";
import Fullscreen from "../../../components/layouts/fullscreen";

export default function ProviderEdit () {
  const router = useRouter()
  const { id } = router.query

  const { data: provider, error } = useSWR(`/api/providers/${id}`)

  console.log(provider)
  
  return (
    <>{id}</>
  )
  
}

ProviderEdit.layout = page => <Fullscreen closeHref='/providers'>{page}</Fullscreen>