import type { Image } from "../../App";

interface IProps {
    images: Image[];
    selectedId: number;
    onClose: () => void;
    onNavigate: (id: number) => void;
}

export default function LightBox({
    images,
    selectedId,
    onClose,
    onNavigate
}: IProps) {
    console.log(images, selectedId, onClose, onNavigate)



    return <img src={`api/images/${selectedId}/full`} />
}